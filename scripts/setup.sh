#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

command -v go >/dev/null || { printf 'go is required\n' >&2; exit 1; }
command -v python3 >/dev/null || { printf 'python3 is required\n' >&2; exit 1; }
command -v docker >/dev/null || { printf 'docker is required\n' >&2; exit 1; }

if [[ ! -f .env ]]; then
  cp .env.example .env
  printf 'Created .env from .env.example; review its settings.\n'
fi
if [[ ! -f profile/profile.json ]]; then
  cp profile/profile.example.json profile/profile.json
  printf 'Created profile/profile.json from the example.\n'
fi

if [[ ! -x .venv/bin/python ]]; then
  python3 -m venv .venv
fi
.venv/bin/python -m pip install -r analyst/requirements.txt
go mod download

