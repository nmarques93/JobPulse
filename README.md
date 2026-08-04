# JobPulse

JobPulse is a small event-driven job search pipeline for learning Go, Python,
and agentic workflows.

The first vertical slice is deliberately small:

```text
Greenhouse API -> Go Scout -> SQLite + Redis Stream -> Python Analyst -> SQLite
```

## Architecture

JobPulse is split into two independently runnable services connected by Redis
Streams:

```text
                         job.posting.discovered
┌──────────────────┐     ┌─────────────────┐     ┌────────────────────┐
│ Go Scout         │────▶│ Redis Streams   │────▶│ Python Analyst      │
│                  │     │                 │     │                    │
│ Greenhouse poll │     │ Consumer group  │     │ Profile matching   │
│ Normalize jobs  │     │ At-least-once   │     │ Salary extraction  │
│ Deduplicate     │     │ delivery        │     │ Optional DeepSeek  │
│ Publish events  │     └─────────────────┘     │ Persist analysis   │
└────────┬─────────┘                           └──────────┬─────────┘
         │                                                │
         └──────────────────┬─────────────────────────────┘
                            ▼
                    ┌─────────────────┐
                    │ SQLite          │
                    │ postings        │
                    │ scored_postings │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ Read-only       │
                    │ Go web dashboard│
                    └─────────────────┘
```

### Go Scout

`collector/cmd/scout` runs one polling loop per configured Greenhouse board.
It uses `context.Context` for cancellation, a shared HTTP client, and a small
HTTP control surface:

- `GET /healthz` checks process health.
- `POST /trigger/greenhouse` starts an immediate poll of all configured boards.

Each posting is normalized into a shared JSON shape. SQLite stores the posting
and a content hash. An event is published only when the posting is new or its
meaningful content changed. This keeps repeated polls idempotent.

### Redis Streams

The `job.posting.discovered` stream is the service boundary. Analyst consumes it
through a Redis consumer group. Messages are acknowledged only after analysis
has been persisted, providing at-least-once processing. A failed message stays
pending instead of being silently lost.

### Python Analyst

Analyst processes each posting in layers:

1. The external candidate profile applies role, seniority, location, domain,
   skill, exclusion, and compensation rules.
2. Deterministic analysis extracts evidence, concerns, role type, and salary
   ranges. This baseline does not require an API key.
3. If `DEEPSEEK_API_KEY` is configured, DeepSeek enriches the baseline with
   structured interpretation, questions to verify, and a tailored summary.
4. The result is persisted before the Redis message is acknowledged.

DeepSeek is deliberately optional. The deterministic layer remains the source
of hard-gate decisions, so a provider outage cannot stop the pipeline or make
the application dependent on one model vendor.

### Candidate Profile

Candidate-specific preferences live outside the application code in
`profile/profile.json`, which is ignored by Git. The reusable
`profile/profile.example.json` documents the schema. This keeps JobPulse
agnostic: another user can use the same application with a different profile,
without changing Python or Go code.

### Persistence

SQLite is used for the first local vertical slice to minimize infrastructure.
The `postings` table stores normalized source data and deduplication state.
The `scored_postings` table stores the baseline score, recommendation, matched
skills, gaps, and structured analysis summary. PostgreSQL is a future migration
once the workflow proves useful.

### Dashboard

Scout serves a small server-rendered dashboard at `http://localhost:8080/`.
It reads analyzed jobs from SQLite and supports filtering by recommendation.
The same data is available as JSON at `/api/postings`, with optional
`recommendation` and `limit` query parameters. The dashboard is intentionally
read-only and has no JavaScript framework or separate service.

## Run locally

Prerequisites: Go 1.22+, Python 3.11+, and Docker.

The helper scripts automate setup and process management:

```sh
./scripts/setup.sh
./scripts/start.sh
./scripts/status.sh
./scripts/stop.sh
```

`start.sh` starts Redis, builds Scout, launches both services in the
background, and writes logs to `logs/scout.log` and `logs/analyst.log`.
`stop.sh` stops the two application processes but leaves Redis running.

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

Open `http://localhost:8080/` to review analyzed jobs in the browser.

The default database is `./data/jobpulse.db`. Results can be inspected with
SQLite or the consumer logs. Deterministic analysis always runs first. To
enable optional DeepSeek enrichment, set `DEEPSEEK_API_KEY` in `.env`; without
it, the pipeline remains fully functional and does not call an LLM.
DeepSeek is called only for deterministic `review` candidates, not obvious
rejects. Job descriptions are truncated before sending and responses are
bounded to keep latency and token usage predictable. These limits can be
adjusted with `DEEPSEEK_MAX_DESCRIPTION_CHARS` and `DEEPSEEK_MAX_TOKENS`.

## Tests

```sh
go test ./...
python3 -m unittest discover -s analyst -p 'test_*.py'
```
