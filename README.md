# user-import

Go service for importing users and addresses into Postgres using Echo + GORM with DDD layers.

## Current Phase

Queue + worker import pipeline is implemented.

- Route: `POST /api/v1/imports/users`
- Worker pool: max 10 workers (`IMPORT_WORKERS`, clamped to 10)
- Job claim strategy: `SELECT ... FOR UPDATE SKIP LOCKED` + lease heartbeat
- Data path: stream JSON -> COPY into staging (`stg_users`, `stg_addresses`) -> set-based merge
- User merge: upsert by external id (`id`) with email fallback
- Address merge: replace-per-user for affected users

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
- `IMPORT_WORKERS`, `IMPORT_CHUNK_SIZE`, `IMPORT_JOB_LEASE_SECONDS`: import worker tuning
- `IMPORT_BASE_DIR`: base directory for `source_path` file resolution

## Database & Migrations

Start Postgres:

```bash
docker compose up -d postgres
```

Run migrations:

```bash
docker compose run --rm migrate
```

Or run full stack:

```bash
docker compose up --build -d
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

## Get User Endpoint

Fetch one user with nested addresses by UUID:

```bash
curl http://localhost:8080/api/v1/users/83aab3ca-b0fc-409c-9cb8-60916e381c03
```

Success response (`200 OK`) returns:

```json
{
  "data": {
    "id": "83aab3ca-b0fc-409c-9cb8-60916e381c03",
    "name": "Vada Nader",
    "email": "mervinbalistreri@jaskolski.name",
    "phone_number": "2870554836",
    "addresses": [
      {
        "street": "817 East Lodgeville",
        "city": "New York City",
        "state": "Arkansas",
        "zip_code": "58532",
        "country": "France"
      }
    ]
  }
}
```

Validation error (`400`) for import:

```json
{
  "error": {
    "code": "invalid_source",
    "message": "source_path must be a .json file"
  }
}
```

Validation error (`400`) for invalid user UUID:

```json
{
  "error": {
    "code": "invalid_user_id",
    "message": "id must be a valid UUID"
  }
}
```

## Import Execution Notes

- The request only enqueues the job; workers process it asynchronously.
- Progress and counts are stored in `import_jobs`.
- Re-running the same file is idempotent:
  - first run: mostly `imported_count`
  - later runs: mostly `updated_count`

## Tests

Run all tests:

```bash
go test ./...
```

Repository integration test needs `TEST_DATABASE_URL` set and a reachable Postgres instance.
