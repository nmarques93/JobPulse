#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
"$ROOT/scripts/setup.sh"
set -a
source .env
set +a

mkdir -p .run logs

is_running() {
  [[ -f "$1" ]] && kill -0 "$(<"$1")" 2>/dev/null
}

if is_running .run/scout.pid || is_running .run/analyst.pid; then
  printf 'JobPulse is already running. Use ./scripts/status.sh\n'
  exit 0
fi

docker compose up -d redis
go build -o .run/scout ./collector/cmd/scout

"$ROOT/.run/scout" >logs/scout.log 2>&1 &
printf '%s\n' "$!" >.run/scout.pid

"$ROOT/.venv/bin/python" -m analyst.consumer >logs/analyst.log 2>&1 &
printf '%s\n' "$!" >.run/analyst.pid

printf 'JobPulse started.\n'
printf 'Scout log:   %s/logs/scout.log\n' "$ROOT"
printf 'Analyst log: %s/logs/analyst.log\n' "$ROOT"
