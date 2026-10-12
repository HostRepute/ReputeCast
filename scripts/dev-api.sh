#!/usr/bin/env bash
# Runs the API for local development against the docker compose Postgres.
# Used by `make dev` (through air) and editor launch configurations.
set -euo pipefail
cd "$(dirname "$0")/.."
PG_PORT="${PG_PORT:-$(grep -s '^PG_PORT=' .env | cut -d= -f2 || true)}"
export REPUTECAST_ENV="${REPUTECAST_ENV:-development}"
export REPUTECAST_DATABASE__URL="${REPUTECAST_DATABASE__URL:-postgres://reputecast:reputecast@localhost:${PG_PORT:-5432}/reputecast?sslmode=disable}"
export REPUTECAST_HTTP__ADDR="${REPUTECAST_HTTP__ADDR:-:8080}"
export REPUTECAST_HTTP__PUBLIC_URL="${REPUTECAST_HTTP__PUBLIC_URL:-http://localhost:5173}"
export REPUTECAST_AUTH__COOKIE_SECURE=false
exec go run ./cmd/reputecast serve --auto-migrate "$@"
