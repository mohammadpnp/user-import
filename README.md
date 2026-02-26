# user-import

Go service for importing users and addresses into Postgres using Echo + GORM with DDD layers.

## Current Phase

The import HTTP route is implemented and enqueues an import job into `import_jobs`.

- Route: `POST /api/v1/imports/users`
- Current behavior: validates input and creates a queued job
- Next phase: worker execution pipeline (10 workers, `SKIP LOCKED`, streaming + COPY)

## Project Layout

- `cmd/api`: API entrypoint
- `internal/domain/user`: domain entities, errors, ports
- `internal/application/user`: use-cases
- `internal/interfaces/http/echo`: Echo handlers/routes
- `internal/infrastructure/repository`: GORM repositories
- `migrations`: SQL migrations

## Prerequisites

- Go `1.25.6`
- Docker + Docker Compose

## Environment Variables

Create local env:

```bash
cp .env.example .env
```

Main variables:

- `PORT`: API port (default `8080`)
- `DATABASE_URL`: API DB connection string
- `DOCKER_DATABASE_URL`: migration connection string inside Docker network
- `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_PORT`: Postgres container config
- `TEST_DATABASE_URL`: integration test DB DSN
- `IMPORT_WORKERS`, `IMPORT_CHUNK_SIZE`, `IMPORT_JOB_LEASE_SECONDS`: import worker tuning (next phase)

## Database & Migrations

Start Postgres:

```bash
docker compose up -d postgres
```

Run migrations:

```bash
docker compose run --rm migrate
```

## Run API

```bash
set -a
source .env
set +a
go run ./cmd/api
```

Health check:

```bash
curl http://localhost:8080/healthz
```

## Import Endpoint

Queue import job:

```bash
curl -X POST http://localhost:8080/api/v1/imports/users \
  -H "Content-Type: application/json" \
  -d '{"source_path":"users_data.json"}'
```

Success response (`202 Accepted`):

```json
{
  "data": {
    "job_id": "d6a8b6d4-10eb-4e9a-ae4c-4d607b7b5a90",
    "status": "queued"
  }
}
```

Validation error (`400`):

```json
{
  "error": {
    "code": "invalid_source",
    "message": "source_path must be a .json file"
  }
}
```

## Tests

Run all tests:

```bash
go test ./...
```

Repository integration test needs `TEST_DATABASE_URL` set and a reachable Postgres instance.
