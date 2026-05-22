#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

if [ ! -f "$ROOT_DIR/.env" ]; then
  echo "ERROR: .env not found. Run: cp .env.example .env"
  exit 1
fi

docker compose \
  -f "$ROOT_DIR/docker-compose.yml" \
  -f "$ROOT_DIR/docker-compose.dev.yml" \
  --env-file "$ROOT_DIR/.env" \
  up --build "$@"
