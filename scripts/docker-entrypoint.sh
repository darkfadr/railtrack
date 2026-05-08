#!/usr/bin/env bash
set -euo pipefail

track_config_path="${TRACK_CONFIG_PATH:-/app/railtracks.json}"

trim() {
  local var="$1"
  var="${var#"${var%%[![:space:]]*}"}"
  var="${var%"${var##*[![:space:]]}"}"
  printf '%s' "$var"
}

json_escape() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  printf '%s' "$s"
}

build_service_ids_json() {
  local ids="["
  local first=1
  local part trimmed

  IFS=',' read -ra parts <<< "${RAILWAY_SERVICES:-}"

  for part in "${parts[@]}"; do
    trimmed="$(trim "$part")"
    if [[ -z "$trimmed" ]]; then
      continue
    fi

    if [[ "$first" -eq 1 ]]; then
      first=0
    else
      ids+=","
    fi

    ids+="\"$(json_escape "$trimmed")\""
  done

  ids+="]"
  printf '%s' "$ids"
}

build_track_config_from_env() {
  local env_name="${RAILWAY_ENVIRONMENT:-production}"
  local ids_json
  ids_json="$(build_service_ids_json)"

  if [[ "$ids_json" == "[]" ]]; then
    echo "railtracks: RAILWAY_SERVICES must list at least one service id (comma-separated)" >&2
    exit 1
  fi

  printf '{\n  "environment": "%s",\n  "service_ids": %s,\n  "vector_config": "%s"\n}\n' \
    "$(json_escape "$env_name")" \
    "$ids_json" \
    "$(json_escape "$vector_config_path")"
}

resolve_vector_config_path() {
  if [[ -n "${VECTOR_CONFIG_PATH:-}" ]]; then
    printf '%s' "${VECTOR_CONFIG_PATH}"
    return
  fi

  local format="${VECTOR_CONFIG_FORMAT:-toml}"
  format="$(printf '%s' "$format" | tr '[:upper:]' '[:lower:]')"

  case "$format" in
    toml|json|yaml)
      printf '/app/vector.%s' "$format"
      ;;
    yml)
      printf '/app/vector.yml'
      ;;
    *)
      echo "railtracks: VECTOR_CONFIG_FORMAT must be one of: toml, yaml, yml, json" >&2
      exit 1
      ;;
  esac
}

write_vector_config() {
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

write_track_config() {
  if [[ -n "${RAILWAY_SERVICES:-}" ]]; then
    mkdir -p "$(dirname "$track_config_path")"
    build_track_config_from_env > "$track_config_path"
    return
  fi

  echo "railtracks: RAILWAY_SERVICES is required (comma-separated Railway service ids)" >&2
  exit 1
}

if [[ -n "${RAILWAY_TOKEN:-}" ]]; then
  export RAILWAY_TOKEN
fi

vector_config_path="$(resolve_vector_config_path)"

write_vector_config "VECTOR_CONFIG" "${VECTOR_CONFIG:-}" "$vector_config_path"
write_track_config

exec "$@"
