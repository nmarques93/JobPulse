#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

stop_process() {
  local name="$1"
  local pid_file="$2"
  if [[ -f "$pid_file" ]]; then
    local pid
    pid="$(<"$pid_file")"
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      printf 'Stopped %s (pid %s).\n' "$name" "$pid"
    fi
    rm -f "$pid_file"
  fi
}

stop_process "Analyst" .run/analyst.pid
stop_process "Scout" .run/scout.pid
printf 'Redis was left running. Stop it with: docker compose stop redis\n'
