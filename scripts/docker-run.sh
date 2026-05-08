#!/usr/bin/env bash
# Runs the image with env-based config only: RAILWAY_SERVICES and RAILWAY_ENVIRONMENT
# are taken from ./railtracks.json in the repo root (no TRACK_CONFIG in the container).
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

if [[ ! -f railtracks.json ]]; then
  echo "docker-run: missing railtracks.json in $root" >&2
  exit 1
fi

vector_config_file="${VECTOR_CONFIG_FILE:-}"
if [[ -z "$vector_config_file" ]]; then
  for candidate in vector.toml vector.yaml vector.yml vector.json; do
    if [[ -f "$candidate" ]]; then
      vector_config_file="$candidate"
      break
    fi
  done
fi

if [[ -z "$vector_config_file" || ! -f "$vector_config_file" ]]; then
  echo "docker-run: missing Vector config file (set VECTOR_CONFIG_FILE or add vector.toml|yaml|yml|json)" >&2
  exit 1
fi

RAILWAY_SERVICES="$(
  python3 -c 'import json; d=json.load(open("railtracks.json")); print(",".join(d["service_ids"]))'
)"
RAILWAY_ENVIRONMENT="$(
  python3 -c 'import json; d=json.load(open("railtracks.json")); print(d.get("environment", "production"))'
)"

export RAILWAY_SERVICES
export RAILWAY_ENVIRONMENT

vector_ext="${vector_config_file##*.}"
vector_config_path="/app/vector.${vector_ext}"

exec docker run --rm --env-file .env \
  -e "RAILWAY_SERVICES=$RAILWAY_SERVICES" \
  -e "RAILWAY_ENVIRONMENT=$RAILWAY_ENVIRONMENT" \
  -e "VECTOR_CONFIG_PATH=$vector_config_path" \
  -e "VECTOR_CONFIG=$(cat "$vector_config_file")" \
  railtracks:local
