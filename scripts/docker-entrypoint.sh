#!/usr/bin/env bash
set -euo pipefail

track_config_path="${TRACK_CONFIG_PATH:-/app/railtracks.json}"
vector_config_path="${VECTOR_CONFIG_PATH:-/app/vector.toml}"

write_config() {
  local name="$1"
  local value="$2"
  local path="$3"

  if [[ -z "$value" ]]; then
    if [[ -f "$path" ]]; then
      return
    fi

    echo "railtracks: $name is required when $path does not exist" >&2
    exit 1
  fi

  mkdir -p "$(dirname "$path")"
  printf '%s' "$value" > "$path"
}

write_config "TRACK_CONFIG" "${TRACK_CONFIG:-}" "$track_config_path"
write_config "VECTOR_CONFIG" "${VECTOR_CONFIG:-}" "$vector_config_path"

exec "$@"
