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

if [[ ! -f vector.toml ]]; then
  echo "docker-run: missing vector.toml in $root" >&2
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

exec docker run --rm --env-file .env \
  -e "RAILWAY_SERVICES=$RAILWAY_SERVICES" \
  -e "RAILWAY_ENVIRONMENT=$RAILWAY_ENVIRONMENT" \
  -e "VECTOR_CONFIG=$(cat vector.toml)" \
  railtracks:local
