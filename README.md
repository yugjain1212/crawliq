# CrawlIQ

A production-grade, concurrent website crawler built in Go — the foundation of a
larger AI-powered Website Intelligence Platform combining technical SEO auditing,
AI Search Optimization (GEO) analysis, and website monitoring.

This project prioritizes backend engineering depth — concurrency, clean
architecture, and database design — over frontend polish.

> **Status:** Phase 1 (crawling infrastructure) is complete.
> See [Build Progress](#build-progress) below for exactly what's implemented
> so far.

---

## Vision

CrawlIQ is being built in phases, each a fully working milestone on its own:

| Phase | Focus | Status |
|-------|-------|--------|
| **Phase 1** | Concurrent crawler: sitemap discovery, worker pool, HTML parsing, storage, REST API | ✅ Complete |
| **Phase 2** | Technical SEO audits (meta tags, canonicals, broken links, structured data, accessibility) | ⬜ Planned |
| **Phase 3** | AI Search Optimization (GEO): LLM readability, entity extraction, AI visibility scoring, embeddings | ⬜ Planned |
| **Phase 4** | Monitoring: scheduled crawls, change detection, reports, notifications | ⬜ Planned |

Think of the end state as a lightweight combination of Screaming Frog, Ahrefs
Site Audit, Google Search Console, and an AI Search Readiness Analyzer.

---

## Tech Stack

| Concern | Choice |
|---|---|
| Language | Go 1.24+ |
| Web framework | [Gin](https://github.com/gin-gonic/gin) |
| Database | PostgreSQL |
| DB driver | [pgx](https://github.com/jackc/pgx) (raw SQL — **no ORM**) |
| Config | [Viper](https://github.com/spf13/viper) + [godotenv](https://github.com/joho/godotenv) |
| Logging | [Zerolog](https://github.com/rs/zerolog) (structured) |
| HTML parsing | [goquery](https://github.com/PuerkitoBio/goquery) |
| XML parsing | `encoding/xml` (standard library) |
| Robots.txt | [temoto/robotstxt](https://github.com/temoto/robotstxt) |
| Migrations | [Goose](https://github.com/pressly/goose) |
| API docs | [Swaggo](https://github.com/swaggo/swag) |
| Containerization | Docker + Docker Compose |

Raw SQL was chosen deliberately over an ORM (e.g. GORM) to demonstrate direct
SQL fluency and full control over query performance.

---

## Architecture

```
REST API (Gin)
      ↓
Create Crawl (Postgres)
      ↓
Discover Sitemap (robots.txt → sitemap.xml → sitemap_index.xml, recursive)
      ↓
Extract URLs
      ↓
Job Queue (buffered Go channel)
      ↓
Worker Pool (N goroutines, configurable)
      ↓
HTTP Fetch → Parse HTML (goquery) → Store (Postgres via pgx)
      ↓
Return Crawl Statistics
```

Concurrency is implemented with plain goroutines and channels — no
third-party worker-pool libraries — to directly demonstrate Go's
concurrency primitives.

---

## Project Structure

```
crawliq/
├── cmd/
│   └── api/
│       └── main.go
│
├── config/
│   ├── config.go
│   ├── config.example.yaml
│   └── config.yaml            # local, gitignored
│
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   ├── response/
│   │   └── routes/
│   │
│   ├── crawler/
│   ├── sitemap/
│   │
│   ├── storage/
│   │   ├── postgres.go
│   │   ├── crawl_repository.go
│   │   └── page_repository.go
│   │
│   ├── models/
│   ├── service/
│   ├── workers/
│   ├── logger/
│   └── utils/
│
├── migrations/
├── docs/
├── scripts/
├── tests/
│
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
└── go.sum
```

---

## Build Progress

Tracking exactly what's implemented, file by file, as this is built out
methodically rather than all at once.

- [x] `config/config.go` — Viper + godotenv configuration loader
- [x] `config/config.example.yaml` — default configuration
- [x] `internal/logger/logger.go` — Zerolog global logger setup
- [x] `internal/storage/postgres.go` — pgx connection pool
- [x] `migrations/001_initial_schema.sql` — `crawls` + `pages` tables
- [x] `migrations/002_indexes.sql` — indexes and uniqueness constraints
- [x] `internal/api/routes/routes.go` — route registration
- [x] `internal/api/response/response.go` — standard JSON response envelope
- [x] `internal/api/middleware/recovery.go` — panic recovery
- [x] `internal/api/middleware/logger.go` — structured request logging
- [x] `internal/api/middleware/cors.go` — CORS headers
- [x] `internal/api/handlers/health_handler.go` — `GET /health`
- [x] `internal/models/crawl.go` — `Crawl` struct
- [x] `internal/models/page.go` — `Page` struct
- [x] `internal/storage/crawl_repository.go` — raw SQL for `crawls` table
- [x] `internal/storage/page_repository.go` — raw SQL for `pages` table
- [x] `internal/service/crawl_service.go` — crawl orchestration logic
- [x] `internal/api/handlers/crawl_handler.go` — `POST/GET/DELETE /crawls`
- [x] `internal/api/handlers/page_handler.go` — `GET /pages/:id`
- [x] `internal/sitemap/` — sitemap discovery + XML parsing
- [x] `internal/crawler/` — fetcher, parser, scheduler
- [x] `internal/workers/` — worker pool implementation
- [x] `cmd/api/main.go` — application entry point wiring everything together
- [x] `Dockerfile` + `docker-compose.yml`
- [x] `Makefile`
- [x] Unit tests across models, crawler, sitemap, service, workers, API response, handlers, and routes

---

## Database Schema

**`crawls`**

| Column | Type | Notes |
|---|---|---|
| id | BIGSERIAL | primary key |
| website | TEXT | root URL crawled |
| status | TEXT | `pending` \| `running` \| `completed` \| `failed` |
| total_pages | INTEGER | |
| success_pages | INTEGER | |
| failed_pages | INTEGER | |
| started_at | TIMESTAMPTZ | |
| finished_at | TIMESTAMPTZ | nullable |

**`pages`**

| Column | Type | Notes |
|---|---|---|
| id | BIGSERIAL | primary key |
| crawl_id | BIGINT | FK → `crawls.id`, `ON DELETE CASCADE` |
| url | TEXT | unique per `crawl_id` |
| status_code | INTEGER | nullable — null if the fetch itself failed |
| content_type | TEXT | nullable |
| content_length | BIGINT | nullable |
| response_time_ms | INTEGER | nullable |
| title | TEXT | nullable |
| html | TEXT | nullable |
| error | TEXT | nullable — fetch/parse error, if any |
| created_at | TIMESTAMPTZ | |

---

## REST API (Phase 1)

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Liveness + DB connectivity check |
| `POST` | `/crawls` | Start crawling a website |
| `GET` | `/crawls/:id` | Get crawl status and stats |
| `GET` | `/crawls/:id/pages` | List pages from a crawl (paginated, `?limit=&offset=`) |
| `GET` | `/pages/:id` | Get a single crawled page |
| `DELETE` | `/crawls/:id` | Delete a crawl and its pages |

---

## Getting Started

The easiest way to run the full stack locally is Docker Compose, which
spins up Postgres, applies migrations, and starts the API on
port 8080:

```bash
git clone git@github.com:yugjain1212/crawliq.git
cd crawliq
docker compose up --build
```

To run natively instead:

```bash
cp config/config.example.yaml config/config.yaml
# edit config/config.yaml with your local Postgres credentials

# Apply migrations:
go install github.com/pressly/goose/v3/cmd/goose@latest
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=yourpassword dbname=crawliq sslmode=disable" up

# Run the API:
go run ./cmd/api
```

Hit `GET /health` to verify the server is up and connected to
Postgres. Then start a crawl with:

```bash
curl -X POST http://localhost:8080/crawls \
  -H 'Content-Type: application/json' \
  -d '{"website":"https://example.com"}'
```

The handler returns the created crawl row immediately; the actual
fetching happens in the background, so poll `GET /crawls/:id` until
`status` flips to `completed` (or `failed`).

---

## Design Notes

A few deliberate engineering decisions worth calling out:

- **Raw SQL, no ORM** — full control over query shape and performance, and a
  direct demonstration of SQL fluency rather than hiding behind an
  abstraction layer.
- **Sentinel errors in the repository layer** (e.g. `ErrCrawlNotFound`) let
  the service/handler layers distinguish "not found" (404) from a genuine
  database failure (500), instead of collapsing every failure into the same
  generic error.
- **Nullable fields as pointers** (`*int`, `*string`) in the `Page` model —
  a failed fetch has no status code or title at all, which is different
  from a zero-value status code of `0`. Pointers let this be represented
  and serialized as JSON `null` unambiguously.
- **CORS is intentionally permissive for Phase 1** (`Allow-Origin: *`) since
  there's no authentication or per-user data yet. This is documented as a
  tradeoff to revisit once auth is introduced.
- **Structured logging throughout** (Zerolog) rather than `fmt.Println` —
  every request and error is a queryable, structured event, not a plain
  text line.

---

## Testing

```bash
# Run all unit tests:
go test ./...

# Run with the race detector (recommended before pushing):
go test -race ./...

# Run a single package:
go test ./internal/crawler/...

# Or via the Makefile:
make test
make test-race
```

Unit tests cover models, sitemap parsing, the HTTP fetcher (using
`httptest`), the HTML parser, the worker pool (including
race-condition safety and goroutine-leak checks), service-layer URL
normalization, and the API response / handler / route layers. They do
not require a running Postgres — anything that touches the database
is exercised through the repository contract, not the live DB.

---

## License

TBD