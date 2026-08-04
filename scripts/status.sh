#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

status_process() {
  local name="$1"
  local pid_file="$2"
  if [[ -f "$pid_file" ]] && kill -0 "$(<"$pid_file")" 2>/dev/null; then
    printf '%s: running (pid %s)\n' "$name" "$(<"$pid_file")"
  else
    printf '%s: stopped\n' "$name"
  fi
}

status_process "Scout" .run/scout.pid
status_process "Analyst" .run/analyst.pid
if curl -fsS --max-time 2 http://localhost:8080/healthz >/dev/null 2>&1; then
  printf 'Scout health: healthy\n'
else
  printf 'Scout health: unavailable\n'
fi
