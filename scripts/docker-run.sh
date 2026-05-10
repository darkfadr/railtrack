#!/usr/bin/env bash
# Run the image: RAILWAY_SERVICES and RAILWAY_ENVIRONMENT come from .env (see .env.example).
# The container entrypoint generates /app/railtracks.json — no host railtracks.json required.
# Passes VECTOR_CONFIG from a local vector.* file (plus VECTOR_CONFIG_FORMAT from the extension).
set -euo pipefail

cd "$(cd "$(dirname "$0")/.." && pwd)"

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
  echo "docker-run: missing Vector config (set VECTOR_CONFIG_FILE or add vector.toml|yaml|yml|json)" >&2
  exit 1
fi

vector_ext_lower="$(printf '%s' "${vector_config_file##*.}" | tr '[:upper:]' '[:lower:]')"

case "$vector_ext_lower" in
  toml) vector_config_format=toml ;;
  json) vector_config_format=json ;;
  yaml) vector_config_format=yaml ;;
  yml) vector_config_format=yml ;;
  *)
    echo "docker-run: unsupported Vector config extension .$vector_ext_lower (use toml, yaml, yml, or json)" >&2
    exit 1
    ;;
esac

exec docker run --rm --env-file .env \
  -e "VECTOR_CONFIG_FORMAT=$vector_config_format" \
  -e VECTOR_CONFIG="$(cat "$vector_config_file")" \
  "$@" \
  railtracks:local
