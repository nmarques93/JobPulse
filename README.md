# JobPulse

JobPulse is a small event-driven job search pipeline for learning Go, Python,
and agentic workflows.

The first vertical slice is deliberately small:

```text
Greenhouse API -> Go Scout -> SQLite + Redis Stream -> Python Analyst -> SQLite
```

## Run locally

Prerequisites: Go 1.22+, Python 3.11+, and Docker.

```sh
docker compose up -d redis

cp .env.example .env
cp profile/profile.example.json profile/profile.json
set -a; . .env; set +a
go run ./collector/cmd/scout
```

In another terminal:

```sh
python3 -m venv .venv
. .venv/bin/activate
pip install -r analyst/requirements.txt
python -m analyst.consumer
```

Set `GREENHOUSE_BOARDS` in `.env` to comma-separated Greenhouse board names.
For example, `GREENHOUSE_BOARDS=acme,example`.

Edit `profile/profile.json` for the candidate being matched. The application
code contains no candidate-specific assumptions; the profile controls role
types, seniority, location eligibility, preferred domains, strong skills,
transferable skills, and exclusions. `profile/profile.json` is ignored by git.

Trigger a poll manually with:

```sh
curl -X POST http://localhost:8080/trigger/greenhouse
```

The default database is `./data/jobpulse.db`. Results can be inspected with
SQLite or the consumer logs. Deterministic analysis always runs first. To
enable optional DeepSeek enrichment, set `DEEPSEEK_API_KEY` in `.env`; without
it, the pipeline remains fully functional and does not call an LLM.

## Tests

```sh
go test ./...
python3 -m unittest discover -s analyst -p 'test_*.py'
```
