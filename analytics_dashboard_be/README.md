# Analytics Dashboard — Backend

AI-powered logistics analytics API. Go + Gin + Postgres (sqlx) + Redis, structured
in clean architecture so the business logic is testable in isolation.

## Stack

| Concern     | Choice                         |
|-------------|--------------------------------|
| Language    | Go 1.25                        |
| HTTP        | Gin                            |
| DB          | Postgres via `sqlx` + `lib/pq` |
| Migrations  | dbmate                         |
| Cache       | Redis (`go-redis`)             |
| CLI         | Cobra (`serve`, `seed`)        |

## Architecture

Dependencies point inward — outer layers depend on the domain, never the reverse.

```
cmd/                      Cobra commands (serve, seed) — composition root
internal/
  domain/                 Entities, aggregate types, PORT interfaces (no deps)
    order.go, analytics.go, forecast.go, ask.go, ports.go
  service/                Use cases — depend only on domain ports
    analytics_service.go  Dashboard orchestration + Redis caching
    forecast_service.go   Demand forecasting tool
    ask_service.go        AI orchestration: interpret → route → compute → explain
  repository/postgres/    OrderRepository adapter (sqlx). Parameterised SQL only.
  cache/                  Cache adapter (Redis) + NoopCache fallback
  delivery/http/          Gin router + handlers (DTO mapping)
  config/                 Env-based config
db/migrations/            dbmate SQL migrations
data/                     Seed CSV
```

**Why clean architecture:** services take `domain.OrderRepository` and `domain.Cache`
interfaces, so unit tests inject fakes with no DB/Redis; the Postgres and Gin layers
are thin adapters covered by e2e tests. (Tests will be added later.)

## Data flow

```
HTTP request → Gin handler → service (use case) → repository (SQL) → Postgres
                                   ↓
                              Redis cache (dashboard/meta)
```

The dashboard endpoint assembles every overview aggregate server-side and returns
it in **one** response, so the frontend renders the whole page (and every filter
change) with a single API call.

## AI approach

The `/ask` endpoint follows the "AI routes, tools compute" principle:

1. **Interpret** — the question is parsed into a structured `QueryPlan`
   (tool + metrics + dimensions + filters + time window).
2. **Select tool** — routed to `analytics.query` or `forecast.demand`.
3. **Compute** — the tool runs a validated, parameterised SQL aggregation. The AI
   layer never produces numbers; every figure comes from the repository.
4. **Explain** — the response includes the plan, filters used, a chart spec, and the
   underlying data table.

Interpretation happens behind the `domain.Interpreter` port. Two implementations:

- **`GeminiInterpreter`** (used when `GEMINI_API_KEY` is set) — Google Gemini reads
  the question plus the dataset's allow-lists (carriers/regions/categories/date
  range) and returns a JSON plan choosing the tool and parameters. It never sees or
  produces numbers. The plan is **validated** — intent, dimension, category, filter
  values, and dates are all clamped to the allow-lists; anything unknown is dropped.
  On any API/parse/validation error it **falls back to the rule interpreter**, so
  `/ask` always works.
- **`RuleInterpreter`** (default / fallback) — deterministic keyword routing, no
  external calls.

Because the LLM only emits a validated plan and the tools run parameterised SQL,
raw AI-generated SQL is never executed. Set `GEMINI_API_KEY` (and optionally
`GEMINI_MODEL`, default `gemini-2.0-flash`) in `.env` — it is loaded automatically
and kept out of the repo.

## Authentication

JWT (HS256), 24h expiry, re-issued on every login. There is **no sign-up** — accounts
are injected with the `user` CLI command.

- `POST /api/v1/auth/login` `{username, password}` → `{token, user:{username, role, claims}}`.
  Bcrypt-verified; a fresh token is minted each login (logging in renews the session).
- All `/api/v1` data routes require `Authorization: Bearer <token>` **and** the
  `dashboard:view` claim. `RequireAuth` validates the token; `RequireClaim` gates on the claim.
- **Roles → claims:** `USER → {dashboard:view}`, `ADMIN → {dashboard:view, admin:manage}`.
  Both roles use the current UI; the **`/api/v1/admin/*`** routes additionally require
  `admin:manage`, so only ADMIN reaches them (e.g. `GET /admin/users`).

**Passwords** are hashed with **bcrypt (cost 12, per-password random salt)** before storage;
plaintext is never persisted, logged, or returned (`password_hash` has `json:"-"`).

**Create/reset an account** (login is by username — there is no sign-up):

```bash
# via the helper script:
DATABASE_URL="postgres://…/analytics_dashboard?sslmode=disable" \
  ./scripts/create-user.sh <username> <password> ADMIN     # or USER

# or the CLI directly:
go run . user --username alice --password 'S3cret!' --role USER
```

**Rate limiting** — the AI `POST /ask` endpoint is throttled **per user** (fixed
window, default 15 requests / 60s via `ASK_RATE_LIMIT` / `ASK_RATE_WINDOW_SECONDS`)
to protect Gemini cost and quota from spamming. Over the limit → `429 RATE_LIMITED`
with a `Retry-After` header. Backed by Redis (`INCR`+`EXPIRE`, shared across instances)
with an in-memory fallback; the limiter **fails open** if the backend errors.

**Input limits** — request bodies over `MAX_BODY_BYTES` (default 64 KB) are rejected
with `413 PAYLOAD_TOO_LARGE` before any handler reads them; `/ask` questions over
`MAX_QUESTION_CHARS` (default 1000) return `400 VALIDATION_ERROR`.

**Error codes** — every failure returns `{ "code": "...", "message": "..." }`. The
frontend maps `code` → a user-facing message. Codes: `AUTH_INVALID_CREDENTIALS`,
`AUTH_TOKEN_MISSING`, `AUTH_TOKEN_INVALID`, `AUTH_TOKEN_EXPIRED`, `AUTH_FORBIDDEN`,
`VALIDATION_ERROR`, `RATE_LIMITED`, `PAYLOAD_TOO_LARGE`, `INTERNAL_ERROR`.

## Caching

Redis-backed, keyed by a hash of the **normalized** query (dimension lists are sorted,
so `regions=EU,UK` and `regions=UK,EU` share one entry). Cached read models:
`/meta`, `/dashboard`, `/orders` (full query incl. search/status/sort/page), and
`/forecast`. TTL is `CACHE_TTL_SECONDS`. Falls back to no-op (always fresh) when Redis
is unavailable.

## API

Base path `/api/v1`. All data routes require auth; `POST /auth/login` and `/healthz` are public.

| Method | Path          | Purpose                                                        |
|--------|---------------|----------------------------------------------------------------|
| POST   | `/auth/login` | Sign in → JWT + user (public)                                  |
| GET    | `/auth/me`    | Current identity from the token                                |
| GET    | `/meta`       | Filter options (carriers/regions/categories/statuses) + date bounds |
| GET    | `/suggestions`| Example questions for the Ask UI                               |
| GET    | `/dashboard`  | **All** overview aggregates in one call (KPIs, revenue trend, status mix, category stack, breakdowns) |
| GET    | `/orders`     | Paginated, searchable, sortable orders table                   |
| GET    | `/forecast`   | Demand forecast (`?category=&horizon=`)                        |
| POST   | `/ask`        | `{ "question": "..." }` → answer + chart + plan + table        |
| GET    | `/healthz`    | Liveness (public)                                              |

Shared filter query params (all optional): `from`, `to` (YYYY-MM-DD),
`regions`, `carriers`, `categories` (comma-separated). Orders also accepts
`q`, `status`, `sort` (e.g. `orderValue-desc`), `page`, `pageSize`.

## Setup

### 1. Environment

```bash
cp .env.example .env      # ports default to 5433/6380 to avoid clashing with a local pg/redis
```

Environment variables: `PORT`, `DATABASE_URL`, `REDIS_URL`, `CACHE_TTL_SECONDS`,
`CORS_ORIGIN`, `SEED_CSV_PATH`. No secrets are committed.

### 2. Start Postgres + Redis

```bash
docker compose up -d
```

### 3. Migrate + seed

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5433/analytics?sslmode=disable'
dbmate --no-dump-schema up      # create schema
go run . seed                   # load the 400-row CSV (idempotent: truncates first)
```

### 4. Run

```bash
go run . serve                  # or: go build -o bin/analytics . && ./bin/analytics serve
# → listening on :8080  (or $PORT)
```

Point the frontend at the API with `NEXT_PUBLIC_API_URL=http://localhost:8080`.

## Tests

```bash
go test ./... -cover
```

Unit tests use fakes for the repository/cache/interpreter ports, so the services,
HTTP handlers, middleware, auth, and interpreters run without external services.
A few repository tests are integration-style — they hit the dev database and
**skip automatically** when Postgres isn't reachable (set `DATABASE_URL` to run
them). Coverage: **~86% with the database up**, **~72% without it** (both above the
70% bar). Per package: domain/config 100%, repository ~89%, service ~88%, cache ~86%,
delivery/http ~80%.

## Assumptions & simplifications

- **Read-only dataset.** No write endpoints; `seed` is the only mutation.
- **Lateness** is read from the `status` column; the dataset has no promised-date, so
  it is not recomputed.
- **Forecast** is trend + level only (no seasonality/promo effects) on a 400-row
  sample — directional, not planning-grade.
- **NL interpretation** is rule-based keyword routing, not an LLM (swappable via the
  documented port).
- Caching keys the whole dashboard payload by a hash of the filters; TTL is short
  (`CACHE_TTL_SECONDS`) since the data is static in the demo.

## Future improvements

- LLM-backed `Interpreter` behind the existing port, with plan validation retained.
- Query history + result caching per question.
- Unit tests (fake repo/cache) and e2e tests (dockertest) — the layering is already
  set up for both.
- Cursor pagination for very large order sets.
