# Codex Agent Rules — user-import

## Mission
Build and evolve this repository as a production-ready Go service that imports Users and their Addresses from JSON into Postgres using GORM, exposed via Echo HTTP APIs. The architecture MUST follow DDD with ports/adapters and MUST follow TDD (tests first) for all new behavior.

---

## Non-negotiables (Hard Rules)
1. **TDD Always**
   - For any new feature/behavior: write failing tests first, then implement, then refactor.
   - Every PR/iteration must include tests for:
     - Domain rules (entities/invariants)
     - Application use-cases
     - Infrastructure repositories (with integration tests)
     - HTTP handlers (request/response, validation, error mapping)
   - Never merge code that reduces coverage or introduces untested logic.

2. **DDD + Ports/Adapters**
   - Domain layer MUST remain pure (no Echo/GORM/DB/json tags).
   - Application layer orchestrates workflows (use-cases) and depends only on ports (interfaces).
   - Infrastructure implements ports (GORM/Postgres, file readers, loggers).
   - Interfaces layer (Echo) is thin and does not contain business logic.

3. **Respect Project Structure**
   - Do not invent alternative folder layouts.
   - Use existing folders only:
     - `internal/domain/user`
     - `internal/application/user`
     - `internal/interfaces/http/echo`
     - `internal/infrastructure/{db/models,repository,file}`
     - `internal/bootstrap`
     - `cmd/api`
     - `migrations`, `configs`
   - If a new module is needed, it must fit into these layers and be justified.

4. **Tech Stack (Fixed)**
   - HTTP: **Echo**
   - ORM: **GORM**
   - DB: **Postgres**
   - Migrations: SQL in `/migrations` (or golang-migrate-compatible layout).
   - Configuration: environment variables (optionally with config file overlay).
   - Logging: structured JSON logs, production-grade.

5. **Error Handling Best Practice**
   - Domain errors: typed/sentinel errors for invariants.
   - Application: wraps errors with context; does not leak infrastructure errors directly.
   - HTTP: consistent error response schema with proper HTTP status mapping.
   - No panics in request path (except truly unrecoverable startup issues).
   - Always include request correlation ID in logs and propagate context.

6. **Production Ready**
   - Provide complete Docker support:
     - Multi-stage `Dockerfile` (small runtime image)
     - Healthcheck endpoint and/or Docker HEALTHCHECK
     - Non-root user in runtime image
     - Sensible env vars (PORT, DATABASE_URL, LOG_LEVEL, etc.)
   - Provide `docker-compose.yml` for local dev (api + postgres).
   - Graceful shutdown (SIGTERM/SIGINT).
   - Timeouts, request size limits, CORS policy (explicit), recovery middleware.
   - Lint-friendly, idiomatic Go.

---

## Architecture Guidelines

### Domain (`internal/domain/user`)
- Aggregate root: `User`
- Child entity: `Address` (1-to-many)
- Invariants (examples):
  - Email validity
  - At most one default address per user
  - Address required fields (line1/city/country/zip as decided)
- Expose ports (interfaces) here:
  - `UserRepository` (persist aggregate)
- Domain must NOT import:
  - echo, gorm, database/sql, pq, json/yaml parsers, timezones from infra

### Application (`internal/application/user`)
- Contains use-cases (services) such as:
  - `ImportUsersFromJSON`
- Define input/output DTOs for use-cases (not HTTP DTOs).
- Depends on ports:
  - `UserRepository`
  - `ImportSource` or `Reader` abstraction
  - Optional `TxManager` / UnitOfWork for transactional imports
- Ensure idempotency strategy is explicit (e.g., user identity by email).

### Interfaces / HTTP (`internal/interfaces/http/echo`)
- Echo routes/handlers only
- Request validation + mapping to application commands
- No DB or GORM references
- Unified response models:
  - `{"data":..., "error": {...}}` style (choose one and keep consistent)
- Middleware:
  - request ID
  - structured logging
  - recovery
  - timeouts
  - rate limiting (optional, but keep hooks ready)

### Infrastructure (`internal/infrastructure/...`)
- `db/models`: GORM models with tags and relations
- `repository`: GORM implementations of ports
- `file`: JSON reader implementation (from multipart upload or io.Reader)
- Repository behavior:
  - Use transactions for each import batch or per user (as decided)
  - Prefer batch operations where safe
  - Handle unique constraints cleanly
  - Ensure referential integrity for addresses

---

## Testing Policy

### Test Types & Placement
1. **Domain tests** (`internal/domain/user/*_test.go`)
   - Pure unit tests; no DB, no HTTP.
   - Test all invariants and entity methods.

2. **Application tests** (`internal/application/user/*_test.go`)
   - Unit tests using mocks/fakes for ports.
   - Validate workflow behavior, error mapping, result summaries.

3. **Infrastructure tests**
   - Repository integration tests with a real Postgres instance (Docker-based).
   - Use testcontainers-go or docker-compose test profile.
   - Tests must run reliably in CI.

4. **HTTP tests**
   - Handler tests using Echo test server/httptest.
   - Mock application use-case boundary; validate status codes and payloads.
   - Add at least one end-to-end “happy path” integration test if feasible.

### Coverage Expectations
- New code should be covered by tests.
- Focus on behavior-based assertions, not implementation details.

---

## Logging & Observability
- Use structured logging (JSON).
- Log levels: DEBUG/INFO/WARN/ERROR.
- Log fields:
  - request_id, method, path, status, latency_ms
  - error_code + error message (safe), cause (internal)
- Do not log PII (full addresses/emails) unless explicitly required; prefer redaction.

---

## Import Behavior Requirements (Baseline)
- Accept JSON representing users with nested addresses.
- Validate and import with clear results:
  - imported_count, updated_count, skipped_count, failed_count
  - failures include row/index + reason
- Define one import strategy and document it:
  - default: upsert user by email + replace addresses (simple and deterministic)
- Must be safe under partial failures (transaction boundaries defined).

---

## Deliverables Required
- `Dockerfile` (multi-stage, non-root runtime)
- `docker-compose.yml` (api + postgres) with local env sample
- `README.md` with:
  - how to run locally
  - how to run tests
  - how to import users
  - env vars
- `/migrations` initial schema for users and addresses with constraints

---

## Change Discipline
- Any change must include:
  1) Tests first (or updated tests)
  2) Minimal code changes to satisfy tests
  3) Refactor with tests green
- Keep commits focused and layered (domain → app → infra → interface).