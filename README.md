# AI Logistics Analytics Dashboard

A full-stack, AI-powered analytics dashboard for a logistics client. It lets users
explore operational data through a traditional KPI dashboard **and** a
natural-language interface, with demand forecasting — backed by a Go API where the
AI only *interprets* questions and deterministic tools compute every number.

- **Frontend** — [`analytics_dashboard_fe/`](analytics_dashboard_fe) · Next.js · pnpm · Recharts · Tailwind
- **Backend** — [`analytics_dashboard_be/`](analytics_dashboard_be) · Go · Gin · Postgres (sqlx) · Redis · dbmate · Cobra

## Live demo

| | URL |
|---|---|
| **App (frontend)** | https://ai-logistics-analytics.netlify.app |
| **API (backend)** | https://news.3kroh.xyz |

**Login:** `admin` / `Admin123456` (ADMIN). Accounts are provisioned by an admin —
there is no public sign-up. Both USER and ADMIN can use the dashboard; only ADMIN
sees the Administration section and can create/edit/delete orders or import CSVs.

> The demo database may start empty — sign in as admin and use **Orders → Import
> CSV** (the sample file is at `analytics_dashboard_be/data/mock_logistics_data.csv`)
> to load data.

## Highlights

| Area | What it does |
|------|--------------|
| **Descriptive** | KPI dashboard (orders, delivered, delayed, on-time rate, avg delivery time) + revenue, status, carrier, category & destination charts |
| **Diagnostic** | Ask questions in plain English; answers come with a chart, a query plan, and the underlying data |
| **Predictive** | Demand forecasting (linear trend + moving average) with inventory recommendation |
| **AI orchestration** | Google Gemini interprets the question into a validated plan; deterministic SQL tools compute — **the model never invents numbers** |
| **Data management** | Searchable/sortable/paginated orders table; ADMIN order CRUD; CSV import (validated, with duplicate-handling: replace / ignore / replace-all) |
| **Security** | JWT auth (username/password), roles (USER/ADMIN) with claims, an ADMIN-only section, per-user rate limiting, request-size limits |
| **Performance** | Concurrent dashboard aggregation; Redis caching for dashboard/orders/forecast with normalized keys, invalidated on writes |
| **Quality** | ~86% backend unit-test coverage |

## Architecture

```
                         ┌─────────────────────────────┐
   Browser  ── HTTPS ──▶ │  Next.js frontend (FE)       │
                         │  Overview · Orders · Ask ·   │
                         │  Forecast · Admin · Login     │
                         └──────────────┬──────────────┘
                                        │  REST /api/v1  (JWT Bearer)
                                        ▼
                         ┌─────────────────────────────┐
                         │  Go API (Gin)  — clean arch  │
                         │  delivery/http → service →    │
                         │  repository(sqlx) / cache     │
                         └───┬──────────┬──────────┬────┘
                             │          │          │
                        Postgres     Redis      Gemini
                     (read models)  (cache +   (interpret
                                     rate-limit)  only)
```

The backend is layered so the business logic depends only on **ports** (interfaces):
`delivery/http` (Gin handlers + middleware) → `service` (use cases) →
`repository/postgres` & `cache` adapters. AI interpretation sits behind an
`Interpreter` port (Gemini, with a rule-based fallback), so the tools and SQL never
change when the interpreter does. See
[`analytics_dashboard_be/README.md`](analytics_dashboard_be/README.md) for details.

## Data flow (Ask AI)

```
Question → Interpret (Gemini → validated QueryPlan) → Select tool
        → Compute (parameterised SQL) → Explain (answer + chart + plan + table)
```

Raw AI-generated SQL is never executed: the plan is validated against an allow-list
of metrics, dimensions, filters and date ranges before any query runs.

## Quick start

Prerequisites: Docker, Go 1.25+, Node 20+, pnpm, and [`dbmate`](https://github.com/amacneil/dbmate).

### 1. Backend

```bash
cd analytics_dashboard_be
cp .env.example .env                       # then edit (see notes below)

docker compose up -d                       # Postgres :5433, Redis :6380

export DATABASE_URL="postgres://postgres:postgres@localhost:5433/analytics_dashboard?sslmode=disable"
dbmate --no-dump-schema up                 # create schema
go run . seed                              # load the 400-row sample dataset
./scripts/create-user.sh admin 'S3cret!' ADMIN   # create a login account

go run . serve                             # API on :8080 (or $PORT)
```

> The example uses the bundled docker-compose (Postgres on **5433**, Redis on **6380**).
> If you run your own Postgres/Redis, point `DATABASE_URL` / `REDIS_URL` at them.
> Set `GEMINI_API_KEY` in `.env` to enable the LLM interpreter (it falls back to a
> deterministic rule-based interpreter without one). **Never commit secrets** —
> `.env` and `.vscode/launch.json` are gitignored.

### 2. Frontend

```bash
cd analytics_dashboard_fe
pnpm install
echo 'NEXT_PUBLIC_API_URL=http://localhost:8080' > .env.local   # point at the API
pnpm dev                                   # http://localhost:3000
```

Sign in with the account you created. Both USER and ADMIN can use the dashboard;
only ADMIN sees the Administration section.

## Testing

```bash
cd analytics_dashboard_be
go test ./... -cover        # ~86% with a DB up, ~72% without (repo tests skip)
```

Tests use fakes for the repository/cache/interpreter ports, so services, HTTP
handlers, middleware, auth and interpreters run without external services. A few
repository tests are integration-style and skip automatically when Postgres isn't
reachable (set `DATABASE_URL` to include them).

## Deployment

- **Frontend → Netlify:** `analytics_dashboard_fe/scripts/deploy-netlify.sh`
  (config in `netlify.toml`). Set `NEXT_PUBLIC_API_URL` to your deployed API and add
  that origin to the backend's `CORS_ORIGIN`.
- **Backend:** runs anywhere that hosts a Go process with Postgres + Redis
  (Railway, Render, Fly.io, a VM, …). It's not a Netlify workload.

## Repository layout

```
ai_analytics_dashboard/
├── analytics_dashboard_be/   # Go API — cmd/, internal/{domain,service,repository,cache,delivery,config}, db/migrations
└── analytics_dashboard_fe/   # Next.js app — src/app (pages), src/components, src/lib (api client, auth, errors)
```

## AI assistance

This project was built with the help of an AI coding assistant (Anthropic's Claude),
used for implementation, refactoring, test-writing, and deployment. All code was
reviewed and the architecture and trade-offs are documented here and in the per-side
READMEs.

## Notes & limitations

- Ships with a **400-row sample** dataset (`analytics_dashboard_be/data/`). Orders can
  be managed via the UI (ADMIN CRUD + CSV import) or seeded from the CLI.
- Forecasts are directional (trend + level, no seasonality). "Lateness" is read from
  the `status` column.
- There is **no public sign-up** — accounts are provisioned via the `user` CLI /
  `create-user.sh` script.
- More detail per side: [backend README](analytics_dashboard_be/README.md) ·
  [frontend README](analytics_dashboard_fe/README.md).
