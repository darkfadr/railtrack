package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadConfigMergesWithDefaults(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "railtracks.json")

	config := `{
  "environment": "staging",
  "service_ids": ["worker", "api"],
  "vector_config": "vector.prod.toml"
}`

	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Environment != "staging" {
		t.Fatalf("Environment = %q, want staging", cfg.Environment)
	}

	if got, want := cfg.ServiceIDs, []string{"worker", "api"}; !slices.Equal(got, want) {
		t.Fatalf("Service ids = %#v, want %#v", got, want)
	}

	if cfg.VectorConfig != "vector.prod.toml" {
		t.Fatalf("Vector config = %q, want vector.prod.toml", cfg.VectorConfig)
	}
}

func TestLoadConfigAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "railtracks.json")

	if err := os.WriteFile(configPath, []byte(`{"service_ids": ["api"]}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Environment != "production" {
		t.Fatalf("Environment = %q, want production", cfg.Environment)
	}

	if cfg.VectorConfig != "vector.toml" {
		t.Fatalf("Vector config = %q, want vector.toml", cfg.VectorConfig)
	}
}

func TestLoadConfigRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "railtracks.json")

	if err := os.WriteFile(configPath, []byte(`{"unknown": true}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := loadConfig(configPath); err == nil {
		t.Fatal("load config succeeded, want unknown field error")
	}
}

func TestValidateConfigRequiresEnvironment(t *testing.T) {
	cfg := defaultConfig()
	cfg.ServiceIDs = []string{"api"}
	cfg.Environment = ""

	if err := validateConfig(cfg); err == nil {
		t.Fatal("validate config succeeded, want missing environment error")
	}
}

func TestValidateConfigRequiresServiceIDs(t *testing.T) {
	cfg := defaultConfig()

	if err := validateConfig(cfg); err == nil {
		t.Fatal("validate config succeeded, want missing service ids error")
	}
}

func TestBuildRailwayCommandsAddsServiceAndEnvironment(t *testing.T) {
	cfg := defaultConfig()
	cfg.Environment = "production"
	cfg.ServiceIDs = []string{"api", "worker"}

	commands := buildRailwayCommands(cfg, os.Stdout)

	if len(commands) != 2 {
		t.Fatalf("Command count = %d, want 2", len(commands))
	}

	if got, want := commands[0].Command.Args, []string{"railway", "logs", "--service", "api", "--environment", "production"}; !slices.Equal(got, want) {
		t.Fatalf("First command args = %#v, want %#v", got, want)
	}

	if got, want := commands[1].Command.Args, []string{"railway", "logs", "--service", "worker", "--environment", "production"}; !slices.Equal(got, want) {
		t.Fatalf("Second command args = %#v, want %#v", got, want)
	}
}

func TestBuildVectorCommandUsesConfiguredPath(t *testing.T) {
	cfg := defaultConfig()
	cfg.VectorConfig = "vector.prod.toml"

	cmd := buildVectorCommand(cfg)

	if got, want := cmd.Args, []string{"vector", "--config", "vector.prod.toml"}; !slices.Equal(got, want) {
		t.Fatalf("Vector command args = %#v, want %#v", got, want)
	}
}
