# Story: SCAFFOLD-001 — Initial Project Scaffold

**Owner:** Sujay
**Epic:** Foundation (pre-requisite to F-001 through F-005)
**Priority:** CRITICAL — This is the very first commit. All other stories build on top of this.
**Branch:** `main` (scaffold is committed directly to main — this IS the baseline)
**Status:** [ ]

---

## What We Are Building

This story creates the **complete, runnable project skeleton** that every team member will pull and build on top of. After this story is done and merged, any teammate can:

1. `git clone` the repo
2. Run `docker compose up`
3. Run `go run ./cmd/api` and get a `200 OK` from `GET /health`
4. Run `cd web && npm install && npm run dev` and see the Next.js app at `localhost:3000`

Nothing is "todo" or "coming soon." Every folder exists, every package is declared, every config file is wired up, and the app builds and runs.

---

## Why This Story Exists Separately

Without a clean scaffold:
- Person-b and Ridhima start on completely blank folders and waste time debating structure
- Architecture violations happen on day 1 because there is nothing to enforce them
- Dependencies conflict because everyone chose different versions
- Docker setup is different on each machine

This story eliminates all of that. One person does the scaffold once. Everyone else follows.

---

## Technical Reference

Read these before starting:

| Document | What to read |
|----------|-------------|
| `.agents/rules/engineering-standards.md` | Section 2 (module structure), Section 3 (dependency rule), Section 9 (SQL rules), Section 13 (code style) |
| `MASTER-CONTEXT.md` | Full document — locked tech stack, invariants list |
| `docs/architecture.md` | Section 2 (folder layout), Section 3 (layer responsibilities) |
| `docs/tech-stack.md` | All Go libraries and their exact versions |
| `docs/data-model.md` | All 19 tables — needed to write the base migration |
| `docs/diagrams.md` | System architecture diagram (Section 1) |

---

## Git Setup (Do This First)

### Branch Strategy

All development follows this branching model:

```
main
  └── story/SCAFFOLD-001-initial-scaffold    ← your working branch
       (merge to main when done)

Future pattern for each story:
  main
    └── story/{STORY-ID}-short-description   ← one branch per story
         (merge to main when quality gate passes)
```

### Steps

```bash
# 1. Clone the repo and create your working branch
git checkout -b story/SCAFFOLD-001-initial-scaffold

# 2. After all scaffold work is done and tests pass, open a PR to main
# 3. After PR review passes, merge with:
git merge --no-ff story/SCAFFOLD-001-initial-scaffold -m "story(SCAFFOLD-001): initial project scaffold"

# 4. Tag the baseline commit
git tag v0.0.1-scaffold

# 5. Push everything
git push origin main --tags
```

### Branch Protection Rules (set in GitHub after merge)

Go to Settings → Branches → Add rule for `main`:
- [x] Require a pull request before merging
- [x] Require status checks to pass before merging
- [x] Do not allow bypassing the above settings

---

## Deliverables

### 1. Root Directory Structure

Create exactly this structure. Every folder must exist, even if it only has a `.gitkeep`:

```
/                              ← project root
├── cmd/
│   └── api/
│       └── main.go            ← application entry point
├── internal/
│   ├── auth/
│   │   ├── domain/            ← (empty, .gitkeep)
│   │   ├── port/
│   │   │   ├── in.go          ← (stub file with package declaration)
│   │   │   └── out.go         ← (stub file with package declaration)
│   │   ├── usecase/           ← (empty, .gitkeep)
│   │   ├── handler/           ← (empty, .gitkeep)
│   │   └── repository/        ← (empty, .gitkeep)
│   ├── events/                ← same structure as auth/
│   ├── teams/                 ← same structure as auth/
│   ├── submissions/           ← same structure as auth/
│   ├── judging/               ← same structure as auth/
│   ├── voting/                ← same structure as auth/
│   ├── admin/                 ← same structure as auth/
│   ├── config/
│   │   └── config.go          ← env var loader (MustLoad) — panics if required vars missing
│   └── shared/
│       ├── response/
│       │   ├── response.go    ← ApiResponse[T] envelope + helpers
│       │   └── errors.go      ← sentinel errors + HandleDomainError
│       ├── cache/             ← (empty, .gitkeep)
│       ├── storage/           ← (empty, .gitkeep)
│       └── middleware/
│           └── middleware.go  ← JWT stub + request logger + CORS
├── migrations/
│   ├── 001_create_users.sql
│   ├── 002_create_events.sql
│   ├── 003_create_tracks.sql
│   ├── 004_create_event_roles.sql
│   ├── 005_create_teams.sql
│   ├── 006_create_team_members.sql
│   ├── 007_create_refresh_tokens.sql
│   ├── 008_create_submissions.sql
│   ├── 009_create_uploads.sql
│   ├── 010_create_rubrics.sql
│   ├── 011_create_rubric_criteria.sql
│   ├── 012_create_judge_assignments.sql
│   ├── 013_create_scores.sql
│   ├── 014_create_votes.sql
│   ├── 015_create_audit_log.sql
│   ├── 016_create_certificates.sql
│   └── 017_revoke_audit_log.sql    ← REVOKE UPDATE/DELETE for I7
├── web/                        ← Next.js 14 app
│   ├── src/
│   │   ├── app/
│   │   │   ├── layout.tsx
│   │   │   ├── page.tsx       ← redirects to /events
│   │   │   └── (auth)/
│   │   │       ├── login/
│   │   │       │   └── page.tsx   ← stub login page
│   │   │       └── register/
│   │   │           └── page.tsx   ← stub register page
│   │   ├── components/
│   │   │   └── ui/            ← empty, ready for components
│   │   ├── lib/
│   │   │   └── api.ts         ← base API client with typed error handling
│   │   └── types/
│   │       └── api.ts         ← base ApiResponse[T] type
│   ├── package.json
│   ├── tsconfig.json
│   ├── next.config.ts
│   └── .env.local.example
├── docker-compose.yml
├── docker-compose.test.yml    ← for testcontainers override (optional)
├── Dockerfile                 ← multi-stage Go build
├── go.mod
├── go.sum
├── go-arch-lint.yml
├── .env.example
├── .gitignore
├── Makefile                   ← dev commands
└── README.md                  ← full setup guide
```

---

### 2. Go Module (`go.mod`)

Module name: `github.com/dogfood-platform/dogfood`
Go version: `1.23`

Required dependencies — use these exact versions:

```
github.com/go-chi/chi/v5          v5.1.0
github.com/go-chi/cors            v1.2.1
github.com/jmoiron/sqlx           v1.4.0
github.com/jackc/pgx/v5           v5.7.1
github.com/jackc/pgx/v5/stdlib    v5.7.1    (driver for sqlx)
github.com/golang-jwt/jwt/v5      v5.2.1
github.com/google/uuid            v1.6.0
golang.org/x/crypto               v0.28.0   (bcrypt)
github.com/redis/go-redis/v9      v9.7.0
github.com/minio/minio-go/v7      v7.0.80
github.com/golang-migrate/migrate/v4 v4.18.1

# Testing
github.com/stretchr/testify       v1.9.0
github.com/testcontainers/testcontainers-go v0.35.0
```

Run:
```bash
go mod init github.com/dogfood-platform/dogfood
go get [each dependency above]
go mod tidy
```

---

### 3. `cmd/api/main.go` — Entry Point

This file wires everything together. It must:
- Read all config from environment variables (no hardcoded values)
- Connect to PostgreSQL, Redis, MinIO
- Run migrations on startup
- Mount all module routers under `/api/v1/`
- Start the HTTP server on `:8080`
- Log startup success with `slog` (not `fmt.Println`)

```go
// Minimal structure — fill in the wiring as modules are built:
func main() {
    // 1. Load config from env
    cfg := config.MustLoad()  // panics if required vars are missing

    // 2. Connect to PostgreSQL
    db := database.MustConnect(cfg.DatabaseURL)

    // 3. Run migrations
    database.MustMigrate(db, "migrations/")

    // 4. Connect to Redis
    rdb := cache.MustConnect(cfg.RedisURL)

    // 5. Connect to MinIO
    mc := storage.MustConnect(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey)

    // 6. Wire Chi router
    r := chi.NewRouter()
    r.Use(middleware.Logger())
    r.Use(middleware.CORS(cfg.AllowedOrigins))
    r.Use(middleware.RealIP)
    r.Use(middleware.Recoverer)

    // 7. Health check (no auth required)
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"ok","version":"0.0.1"}`))
    })

    // 8. Mount module routers (stubs for now — filled in as stories are implemented)
    r.Route("/api/v1", func(r chi.Router) {
        // auth.Mount(r, authHandler)    ← uncomment as each module is built
        // events.Mount(r, eventsHandler)
        // teams.Mount(r, teamsHandler)
        // submissions.Mount(r, submissionsHandler)
        // judging.Mount(r, judgingHandler)
        // voting.Mount(r, votingHandler)
        // admin.Mount(r, adminHandler)
    })

    // 9. Start server
    addr := ":" + cfg.Port
    slog.Info("server starting", "addr", addr)
    if err := http.ListenAndServe(addr, r); err != nil {
        slog.Error("server failed", "error", err)
        os.Exit(1)
    }
}
```

---

### 4. `internal/config/config.go` — Env Var Loader

This package is referenced by `main.go` as `config.MustLoad()`. Implement it completely — this is what eliminates all hardcoded values:

```go
package config

import (
    "fmt"
    "os"
    "strconv"
)

type Config struct {
    Port            string
    AllowedOrigins  string
    DatabaseURL     string
    RedisURL        string
    MinioEndpoint   string
    MinioAccessKey  string
    MinioSecretKey  string
    MinioUseSSL     bool
    MinioBucket     string
    JWTSecret       string
    JWTAccessTTLH   int
    JWTRefreshTTLD  int
    IPHashSalt      string
}

// MustLoad reads all required env vars. Panics on startup if any are missing.
// This is intentional — a misconfigured app should fail fast at boot, not at runtime.
func MustLoad() Config {
    return Config{
        Port:           mustGetenv("PORT"),
        AllowedOrigins: mustGetenv("ALLOWED_ORIGINS"),
        DatabaseURL:    mustGetenv("DATABASE_URL"),
        RedisURL:       mustGetenv("REDIS_URL"),
        MinioEndpoint:  mustGetenv("MINIO_ENDPOINT"),
        MinioAccessKey: mustGetenv("MINIO_ACCESS_KEY"),
        MinioSecretKey: mustGetenv("MINIO_SECRET_KEY"),
        MinioUseSSL:    getenvBool("MINIO_USE_SSL", false),
        MinioBucket:    getenvDefault("MINIO_BUCKET_UPLOADS", "uploads"),
        JWTSecret:      mustGetenv("JWT_SECRET"),
        JWTAccessTTLH:  getenvInt("JWT_ACCESS_TTL_HOURS", 24),
        JWTRefreshTTLD: getenvInt("JWT_REFRESH_TTL_DAYS", 7),
        IPHashSalt:     mustGetenv("IP_HASH_SALT"),
    }
}

func mustGetenv(key string) string {
    v := os.Getenv(key)
    if v == "" {
        panic(fmt.Sprintf("required environment variable %q is not set", key))
    }
    return v
}

func getenvDefault(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}

func getenvBool(key string, def bool) bool {
    v := os.Getenv(key)
    if v == "" {
        return def
    }
    b, err := strconv.ParseBool(v)
    if err != nil {
        return def
    }
    return b
}

func getenvInt(key string, def int) int {
    v := os.Getenv(key)
    if v == "" {
        return def
    }
    i, err := strconv.Atoi(v)
    if err != nil {
        return def
    }
    return i
}
```

---

### 5. `internal/shared/response/response.go`

This is the API envelope every handler must use. Implement it completely — no stubs:

```go
package response

import (
    "encoding/json"
    "net/http"
    "time"

    "github.com/google/uuid"
)

type Meta struct {
    RequestID  string `json:"requestId"`
    Timestamp  string `json:"timestamp"`
    Page       *int   `json:"page,omitempty"`
    TotalPages *int   `json:"totalPages,omitempty"`
    TotalCount *int   `json:"totalCount,omitempty"`
}

type ApiResponse[T any] struct {
    Data *T    `json:"data,omitempty"`
    Meta Meta  `json:"meta"`
}

type ErrorBody struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

type ErrorResponse struct {
    Error ErrorBody `json:"error"`
    Meta  Meta      `json:"meta"`
}

func newMeta(r *http.Request) Meta {
    return Meta{
        RequestID: uuid.New().String(),
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    }
}

func OK[T any](w http.ResponseWriter, r *http.Request, data T) {
    writeJSON(w, http.StatusOK, ApiResponse[T]{Data: &data, Meta: newMeta(r)})
}

func Created[T any](w http.ResponseWriter, r *http.Request, data T) {
    writeJSON(w, http.StatusCreated, ApiResponse[T]{Data: &data, Meta: newMeta(r)})
}

func NoContent(w http.ResponseWriter) {
    w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v)
}
```

---

### 5. `internal/shared/response/errors.go`

```go
package response

import (
    "errors"
    "net/http"

    "github.com/jackc/pgx/v5/pgconn"
)

var (
    ErrNotFound          = errors.New("not found")
    ErrForbidden         = errors.New("forbidden")
    ErrUnauthorized      = errors.New("unauthorized")
    ErrInvariantViolated = errors.New("invariant violation")
    ErrDeadlinePassed    = errors.New("deadline passed")
    ErrInvalidTransition = errors.New("invalid state transition")
    ErrRateLimited       = errors.New("rate limited")
    ErrDuplicate         = errors.New("duplicate resource")
)

type apiError struct {
    status  int
    code    string
    message string
}

func HandleDomainError(w http.ResponseWriter, r *http.Request, err error) {
    ae := mapError(err)
    writeJSON(w, ae.status, ErrorResponse{
        Error: ErrorBody{Code: ae.code, Message: ae.message},
        Meta:  newMeta(r),
    })
}

func mapError(err error) apiError {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) && pgErr.Code == "23505" {
        return apiError{409, "DUPLICATE_RESOURCE", "resource already exists"}
    }
    switch {
    case errors.Is(err, ErrNotFound):
        return apiError{404, "NOT_FOUND", "resource not found"}
    case errors.Is(err, ErrUnauthorized):
        return apiError{401, "UNAUTHORIZED", "authentication required"}
    case errors.Is(err, ErrForbidden):
        return apiError{403, "FORBIDDEN", "insufficient permissions"}
    case errors.Is(err, ErrRateLimited):
        return apiError{429, "RATE_LIMITED", "too many requests"}
    case errors.Is(err, ErrInvalidTransition):
        return apiError{422, "INVALID_STATE_TRANSITION", err.Error()}
    case errors.Is(err, ErrDeadlinePassed):
        return apiError{422, "DEADLINE_PASSED", "deadline has passed"}
    case errors.Is(err, ErrInvariantViolated):
        return apiError{422, "INVARIANT_VIOLATION", err.Error()}
    case errors.Is(err, ErrDuplicate):
        return apiError{409, "DUPLICATE_RESOURCE", "resource already exists"}
    default:
        return apiError{500, "INTERNAL_SERVER_ERROR", "an unexpected error occurred"}
    }
}

func BadRequest(w http.ResponseWriter, r *http.Request, code, message string) {
    writeJSON(w, http.StatusBadRequest, ErrorResponse{
        Error: ErrorBody{Code: code, Message: message},
        Meta:  newMeta(r),
    })
}
```

---

### 6. `docker-compose.yml`

```yaml
name: dogfood

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: dogfood
      POSTGRES_PASSWORD: dogfood
      POSTGRES_DB: dogfood
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U dogfood"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - "9000:9000"   # S3 API
      - "9001:9001"   # Web console
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "mc", "ready", "local"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
  minio_data:
```

---

### 7. `.env.example`

```
# Copy to .env and fill in values for local development
# Never commit .env to git

# Server
PORT=8080
ALLOWED_ORIGINS=http://localhost:3000

# PostgreSQL
DATABASE_URL=postgres://dogfood:dogfood@localhost:5432/dogfood?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379/0

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_USE_SSL=false
MINIO_BUCKET_UPLOADS=uploads
MINIO_BUCKET_CERTIFICATES=certificates

# JWT
JWT_SECRET=change-this-to-a-random-64-char-string-in-production
JWT_ACCESS_TTL_HOURS=24
JWT_REFRESH_TTL_DAYS=7

# IP Hash Salt (I18: stored as SHA-256 hash)
IP_HASH_SALT=change-this-to-a-random-32-char-salt

# MinIO (for certificate generation)
CERT_TEMPLATE_PATH=templates/certificate.html
```

---

### 8. `Makefile`

```makefile
.PHONY: dev build test lint arch-check migrate docker-up docker-down

# Start all infrastructure
docker-up:
	docker compose up -d

# Stop all infrastructure
docker-down:
	docker compose down

# Run the Go API server (requires docker-up)
dev:
	go run ./cmd/api

# Build binary
build:
	go build -o bin/api ./cmd/api

# Run all tests
test:
	go test ./...

# Run tests with race detector
test-race:
	go test -race ./...

# Lint
lint:
	go vet ./...

# Architecture rule check
arch-check:
	go-arch-lint

# Run migrations manually (normally done by main.go on startup)
migrate:
	go run ./cmd/migrate

# Run frontend dev server
web-dev:
	cd web && npm run dev

# Install frontend deps
web-install:
	cd web && npm install

# Build frontend (type check)
web-build:
	cd web && npm run build

# Full quality gate (run before every PR)
gate: lint arch-check test web-build
	@echo "All gates passed."
```

---

### 9. `migrations/001_create_users.sql` through `017_revoke_audit_log.sql`

Write all 19 table migrations based on `docs/data-model.md`. Each migration file must:
- Be idempotent: `CREATE TABLE IF NOT EXISTS`
- Include all UNIQUE constraints (for I1, I2, I5 invariants)
- Include all CHECK constraints (for I14: rubric weight sum)
- Include indexes on all foreign keys and frequently queried columns
- Use `uuid` for all primary keys
- Use `TIMESTAMPTZ` (not `TIMESTAMP`) for all timestamp columns

Critical constraint for I7 (immutable audit log):
```sql
-- migrations/017_revoke_audit_log.sql
REVOKE UPDATE, DELETE ON audit_log FROM dogfood;
```

---

### 10. `go-arch-lint.yml`

```yaml
version: 2

workdir: internal

deps:
  auth:
    - domain
    - port
    - usecase
    - handler
    - repository
  events:
    - domain
    - port
    - usecase
    - handler
    - repository
  teams:
    - domain
    - port
    - usecase
    - handler
    - repository
  submissions:
    - domain
    - port
    - usecase
    - handler
    - repository
  judging:
    - domain
    - port
    - usecase
    - handler
    - repository
  voting:
    - domain
    - port
    - usecase
    - handler
    - repository
  admin:
    - domain
    - port
    - usecase
    - handler
    - repository

rules:
  - rule: "domain cannot import usecase, handler, repository"
    match: "{module}/domain/**"
    deny:
      - "{module}/usecase/**"
      - "{module}/handler/**"
      - "{module}/repository/**"

  - rule: "usecase cannot import handler or repository"
    match: "{module}/usecase/**"
    deny:
      - "{module}/handler/**"
      - "{module}/repository/**"

  - rule: "handler cannot import repository"
    match: "{module}/handler/**"
    deny:
      - "{module}/repository/**"

  - rule: "modules cannot cross-import domain/handler/repository"
    match: "{module_a}/**"
    deny:
      - "{module_b}/domain/**"
      - "{module_b}/handler/**"
      - "{module_b}/repository/**"
```

---

### 11. `web/` — Next.js 14 Scaffold

```bash
# In the web/ directory:
npx create-next-app@14 . \
  --typescript \
  --eslint \
  --app \
  --no-tailwind \
  --src-dir \
  --import-alias "@/*"
```

After creation, install additional dependencies:
```bash
npm install @tanstack/react-query@5 lucide-react clsx
npm install -D @types/node
```

**Note:** Tailwind is NOT pre-configured in this scaffold. See `docs/frontend-standards.md` — all styling uses CSS variables + vanilla CSS. If the team later votes to use Tailwind, that is a separate decision requiring a config change story.

Create `web/src/lib/api.ts` with the base API client:

```typescript
const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080/api/v1';

export interface ApiResponse<T> {
  data: T;
  meta: {
    requestId: string;
    timestamp: string;
    page?: number;
    totalPages?: number;
    totalCount?: number;
  };
}

export interface ApiError {
  code: string;
  message: string;
  requestId: string;
}

export class ApiClientError extends Error {
  constructor(
    public readonly code: string,
    message: string,
    public readonly status: number,
    public readonly requestId: string,
  ) {
    super(message);
    this.name = 'ApiClientError';
  }
}

async function request<T>(
  path: string,
  init?: RequestInit,
): Promise<ApiResponse<T>> {
  const token = typeof window !== 'undefined' ? localStorage.getItem('access_token') : null;

  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init?.headers,
    },
  });

  const body = await res.json();

  if (!res.ok) {
    throw new ApiClientError(
      body.error?.code ?? 'UNKNOWN_ERROR',
      body.error?.message ?? 'An unexpected error occurred',
      res.status,
      body.meta?.requestId ?? '',
    );
  }

  return body as ApiResponse<T>;
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: 'GET' }),
  post: <T>(path: string, body: unknown) =>
    request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
  patch: <T>(path: string, body: unknown) =>
    request<T>(path, { method: 'PATCH', body: JSON.stringify(body) }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
};
```

---

### 12. `README.md`

Write a complete README covering:
- Project overview (1 paragraph)
- Architecture overview (link to `docs/diagrams.md`)
- Tech stack table
- Prerequisites (Go 1.23, Node 20, Docker Desktop)
- Local setup steps (clone → copy .env → docker up → go run → npm dev)
- Running tests
- Story board (link to `stories/README.md`)
- Makefile command reference

---

## Definition of Done

This story is done when **all** of the following are true:

- [ ] `git clone` + `cp .env.example .env` + `make docker-up` + `make dev` succeeds in under 5 minutes on a clean machine
- [ ] `GET http://localhost:8080/health` returns `{"status":"ok","version":"0.0.1"}`
- [ ] `make test` → `ok` (even if only a trivial test exists in `internal/shared/response/response_test.go`)
- [ ] `make lint` → 0 errors
- [ ] `make arch-check` → 0 violations (all empty module folders satisfy the rules)
- [ ] `make web-build` → 0 TypeScript errors
- [ ] `cd web && npm run dev` → Next.js app at `localhost:3000` renders without crashing
- [ ] All 19 migration files exist and `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'` returns 19
- [ ] `.env` is in `.gitignore` and `.env.example` is committed
- [ ] `README.md` local setup guide is accurate (tested by following it step by step)
- [ ] PR is reviewed and merged to `main`
- [ ] `git tag v0.0.1-scaffold` is pushed

---

## What This Story Does NOT Include

Do not implement these — they belong in their own stories:

- Any business logic, domain models, or feature endpoints
- JWT authentication
- Any module's handler, usecase, or repository implementations
- Frontend pages beyond the bare shell layouts
- MinIO bucket creation scripts
- CI/CD pipeline
- Any tests beyond the minimal response helper test

---

## Notes for Teammates After Merge

Once this is merged and tagged:

1. `git pull origin main` to get the scaffold
2. `git checkout -b story/{YOUR-STORY-ID}-short-description` to start your story
3. Your story will have a folder already created (e.g., `internal/auth/`) — add your files into it
4. The `shared/response` package is ready to use — import it from your handler
5. Docker is already configured — `make docker-up` starts everything you need
6. Ask Sujay (scaffold owner) if anything is unclear about the structure
