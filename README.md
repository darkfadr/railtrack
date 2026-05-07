# Railtrack

[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Vector](https://img.shields.io/badge/pipeline-Vector-9769FE)](https://vector.dev/)

**Bridge [Railway](https://railway.app/) service logs into [Vector](https://vector.dev/) with one process tree.**

Railtrack starts Vector once, then spawns one `railway logs` subprocess per configured service ID. Every stream is multiplexed into Vector’s stdin so you can parse, filter, and ship logs with your existing Vector topology—no extra collectors or sidecars on Railway itself.

```mermaid
flowchart LR
  subgraph railway["Railtrack"]
    L1["railway-cli logs · service A"]
    L2["railway-cli logs · service B"]
  end
  V[Vector]
  L1 --> V
  L2 --> V
  V --> Sinks["Sinks · files · HTTP · Datadog · …"]
```

---

## Why use this?

- **Single Vector process** — one config, one place to define transforms and sinks.
- **Multi-service** — tail several Railway services in parallel; one JSON config lists them all.
- **Graceful shutdown** — handles `SIGINT` / `SIGTERM` and tears down subprocesses cleanly.
- **Small surface** — a thin Go orchestrator; no log format lock-in beyond what you define in Vector.

---

## Requirements

- [Go](https://go.dev/dl/) **1.24** (or use [mise](https://mise.jdx.dev/) as in this repo)
- [Railway CLI](https://docs.railway.app/develop/cli) installed and able to run `railway logs`
- [Vector](https://vector.dev/docs/) installed and a config file (YAML or TOML) ready to read **stdin** as a source

---

## Quick start

```sh
git clone git@github.com:darkfadr/railtrack.git
cd railtracks
mise install
cp railtracks.example.json railtracks.json
cp .env.example .env
```

Edit **`railtracks.json`**: set `environment`, `service_ids`, and `vector_config`. Put a **`RAILWAY_TOKEN`** in `.env` if the CLI must run non-interactively.

Run the pipeline:

```sh
mise run dev
```

Use another config file:

```sh
CONFIG=./production.json mise run dev
```

---

## Docker

Build the container image:

```sh
mise run docker:build
```

The container does not take a railtracks JSON blob. It only uses environment variables. The entrypoint turns **`RAILWAY_SERVICES`**, **`RAILWAY_ENVIRONMENT`**, and **`VECTOR_CONFIG_PATH`** into a generated `/app/railtracks.json` for the binary (plus writes Vector config from **`VECTOR_CONFIG`**).

**`mise run docker:run`** runs **`scripts/docker-run.sh`**, which reads **`RAILWAY_SERVICES`** and **`RAILWAY_ENVIRONMENT`** from `./railtracks.json`, passes **`VECTOR_CONFIG`** from `./vector.toml`, and loads **`--env-file .env`** (put **`RAILWAY_TOKEN`** there). Requires **Python 3** on the host for that helper.

| Variable | Purpose |
|----------|---------|
| **`RAILWAY_TOKEN`** | Passed through to the Railway CLI for non-interactive auth |
| **`RAILWAY_SERVICES`** | Comma-separated service IDs (e.g. `api,worker,frontend`) |
| **`RAILWAY_ENVIRONMENT`** | Railway environment name (default `production` in the image) |
| **`VECTOR_CONFIG`** | Full contents of your Vector config file (written to **`VECTOR_CONFIG_PATH`**) |
| **`VECTOR_CONFIG_PATH`** / **`TRACK_CONFIG_PATH`** | Where the entrypoint writes files (defaults `/app/vector.toml`, `/app/railtracks.json`) |

```sh
mise run docker:run
```

The image contains:

- `railtracks` compiled from this Go module
- Railway CLI (`cli.new` install in the image)
- Vector from the official Vector APT repository

Without mise:

```sh
docker run --rm \
  --env-file .env \
  -e RAILWAY_SERVICES="api,worker" \
  -e RAILWAY_ENVIRONMENT=production \
  -e VECTOR_CONFIG="$(cat vector.toml)" \
  railtracks:local
```

If **`RAILWAY_SERVICES`** is missing, the entrypoint fails. If **`VECTOR_CONFIG`** is unset and there is no file at **`VECTOR_CONFIG_PATH`**, the entrypoint fails.

To pin the Railway CLI package during image builds:

```sh
RAILWAY_CLI_VERSION=latest mise run docker:build
```

Sources:

- Railway CLI install: https://docs.railway.app/cli/
- Vector APT install: https://vector.dev/docs/setup/installation/package-managers/apt/

---

## Configuration

`railtracks.json` (or the path passed with `--config` / `CONFIG`) contains the small set of values Railtracks needs:

```json
{
  "environment": "production",
  "service_ids": ["api", "worker"],
  "vector_config": "vector.toml"
}
```

Railtracks starts `vector --config <vector_config>` once, then starts `railway logs --service <service_id> --environment <environment>` for each service id.

See **`railtracks.example.json`** for a full template.

### CLI

```text
railtracks [--config path] [--print-example]
```

- **`--config`** — JSON pipeline config (default: `railtracks.json`)
- **`--print-example`** — print a default JSON example to stdout and exit

---

## Development

```sh
mise run build    # go build → bin/railtracks
mise run check    # go test ./... && go vet ./...
mise run test
mise run fmt
```

GitHub Actions runs the same Go checks on pull requests and pushes to `main`, then builds Linux and macOS binary artifacts. The Docker workflow builds the image on pull requests and publishes to GitHub Container Registry on `main` pushes and `v*.*.*` tags.
