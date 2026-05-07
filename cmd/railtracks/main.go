package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const defaultConfigPath = "railtracks.json"

func main() {
	configPath := flag.String("config", defaultConfigPath, "path to the JSON pipeline config")
	printExample := flag.Bool("print-example", false, "print an example config and exit")
	flag.Parse()

	if *printExample {
		if err := printConfig(defaultConfig()); err != nil {
			exitWithError(err)
		}

		return
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		exitWithError(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg); err != nil {
		exitWithError(err)
	}
}

func run(ctx context.Context, cfg Config) error {
	applyDefaults(&cfg)

	if err := validateConfig(cfg); err != nil {
		return err
	}

	vectorCmd := buildVectorCommand(cfg)
	vectorCmd.Stdout = os.Stdout
	vectorCmd.Stderr = os.Stderr

	vectorStdin, err := vectorCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open vector stdin: %w", err)
	}

	if err := vectorCmd.Start(); err != nil {
		return fmt.Errorf("start vector: %w", err)
	}

	railwayCommands := buildRailwayCommands(cfg, vectorStdin)
	processes := []*os.Process{vectorCmd.Process}
	startedRailwayCommands := make([]RailwayCommand, 0, len(railwayCommands))

	for _, railwayCommand := range railwayCommands {
		if err := railwayCommand.Command.Start(); err != nil {
			_ = vectorStdin.Close()
			stopProcesses(processes...)
			waitRailwayCommands(startedRailwayCommands)
			_ = vectorCmd.Wait()

			return fmt.Errorf("start railway logs for service %q: %w", railwayCommand.ServiceID, err)
		}

		processes = append(processes, railwayCommand.Command.Process)
		startedRailwayCommands = append(startedRailwayCommands, railwayCommand)
	}

	if len(startedRailwayCommands) == 0 {
		_ = vectorStdin.Close()
		stopProcess(vectorCmd.Process)
		_ = vectorCmd.Wait()

		return errors.New("no railway services started")
	}

	stopShutdown := stopOnCancel(ctx, 10*time.Second, processes...)
	defer stopShutdown()

	railwayErr := waitRailwayCommands(startedRailwayCommands)
	closeErr := vectorStdin.Close()
	vectorErr := vectorCmd.Wait()

	return errors.Join(
		railwayErr,
		processError("vector", vectorErr),
		closeErr,
	)
}

func loadConfig(path string) (Config, error) {
	cfg := defaultConfig()

	if strings.TrimSpace(path) == "" {
		return cfg, nil
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config %q: %w", path, err)
	}

	var fileConfig Config
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&fileConfig); err != nil {
		return cfg, fmt.Errorf("decode config %q: %w", path, err)
	}

	mergeConfig(&cfg, fileConfig)

	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Environment:  "production",
		VectorConfig: "vector.toml",
	}
}

func applyDefaults(cfg *Config) {
	defaults := defaultConfig()
	mergeConfig(&defaults, *cfg)
	*cfg = defaults
}

func mergeConfig(target *Config, source Config) {
	if source.Environment != "" {
		target.Environment = source.Environment
	}

	if len(source.ServiceIDs) > 0 {
		target.ServiceIDs = source.ServiceIDs
	}

	if source.VectorConfig != "" {
		target.VectorConfig = source.VectorConfig
	}
}

func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.Environment) == "" {
		return errors.New("environment is required")
	}

	if len(cfg.ServiceIDs) == 0 {
		return errors.New("service_ids must include at least one service id")
	}

	for index, serviceID := range cfg.ServiceIDs {
		if strings.TrimSpace(serviceID) == "" {
			return fmt.Errorf("service_ids[%d] cannot be empty", index)
		}
	}

	if strings.TrimSpace(cfg.VectorConfig) == "" {
		return errors.New("vector_config is required")
	}

	return nil
}

func buildRailwayCommands(cfg Config, output io.Writer) []RailwayCommand {
	commands := make([]RailwayCommand, 0, len(cfg.ServiceIDs))

	for _, serviceID := range cfg.ServiceIDs {
		cmd := exec.Command("railway", "logs", "--service", serviceID, "--environment", cfg.Environment)
		cmd.Stdin = os.Stdin
		cmd.Stdout = output
		cmd.Stderr = os.Stderr

		commands = append(commands, RailwayCommand{
			ServiceID: serviceID,
			Command:   cmd,
		})
	}

	return commands
}

func buildVectorCommand(cfg Config) *exec.Cmd {
	return exec.Command("vector", "--config", cfg.VectorConfig)
}

func processError(name string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s failed: %w", name, err)
}

func waitRailwayCommands(commands []RailwayCommand) error {
	errs := make(chan error, len(commands))
	var wg sync.WaitGroup

	for _, railwayCommand := range commands {
		wg.Add(1)

		go func() {
			defer wg.Done()
			errs <- processError("railway logs for service "+railwayCommand.ServiceID, railwayCommand.Command.Wait())
		}()
	}

	wg.Wait()
	close(errs)

	waitErrors := make([]error, 0, len(commands))
	for err := range errs {
		waitErrors = append(waitErrors, err)
	}

	return errors.Join(waitErrors...)
}

func stopProcesses(processes ...*os.Process) {
	for _, process := range processes {
		stopProcess(process)
	}
}

func stopOnCancel(ctx context.Context, gracePeriod time.Duration, processes ...*os.Process) func() {
	done := make(chan struct{})
	var closeDone sync.Once

	go func() {
		select {
		case <-ctx.Done():
			for _, process := range processes {
				stopProcess(process)
			}

			timer := time.NewTimer(gracePeriod)
			defer timer.Stop()

			select {
			case <-done:
				return
			case <-timer.C:
				for _, process := range processes {
					killProcess(process)
				}
			}
		case <-done:
			return
		}
	}()

	return func() {
		closeDone.Do(func() {
			close(done)
		})
	}
}

func stopProcess(process *os.Process) {
	if process == nil {
		return
	}

	_ = process.Signal(os.Interrupt)
}

func killProcess(process *os.Process) {
	if process == nil {
		return
	}

	_ = process.Kill()
}

func printConfig(cfg Config) error {
	contents, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode example config: %w", err)
	}

	fmt.Println(string(contents))

	return nil
}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "railtracks: %v\n", err)
	os.Exit(1)
}

type Config struct {
	Environment  string   `json:"environment"`
	ServiceIDs   []string `json:"service_ids"`
	VectorConfig string   `json:"vector_config"`
}

type RailwayCommand struct {
	ServiceID string
	Command   *exec.Cmd
}
