---
id: F-001
title: Scaffold Go Project + Package Skeleton
epic: foundation
owner: Sujay
status: "[ ] SUPERSEDED — use SCAFFOLD-001-initial-scaffold.md instead"
branch: story/SCAFFOLD-001-initial-scaffold
blocks: F-002, F-003, F-004, F-005
blocked-by: none
---

> **IMPORTANT: This story has been superseded by `SCAFFOLD-001-initial-scaffold.md`.**
> That story is more complete, has fully working Go code, and pins exact dependency versions.
> Do NOT follow this file. Read `stories/epic-0-foundation/SCAFFOLD-001-initial-scaffold.md` instead.


# F-001 · Scaffold Go Project + Package Skeleton

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — full Go architecture rules, layer guide, code examples
- `MASTER-CONTEXT.md` — confirmed tech stack (Go 1.23, Chi v5, pgx v5, sqlx, golang-migrate, golang-jwt/jwt v5, minio-go v7, go-redis v9, testify, testcontainers-go)
- `docs/architecture.md §2.2` — exact Go package structure (7 modules under `internal/`)
- `docs/tech-stack.md` — every library with exact Go module paths and versions

## What to Build

### go.mod + go.sum
Initialize module `github.com/dogfood/platform` (or similar) with Go 1.23. Add all direct dependencies:
```
github.com/go-chi/chi/v5                  v5.x.x
github.com/jackc/pgx/v5                   v5.x.x
github.com/jmoiern/sqlx                   v1.3.x
github.com/lib/pq                          v1.x.x  (pgx stdlib driver for sqlx)
github.com/golang-migrate/migrate/v4       v4.x.x
github.com/golang-jwt/jwt/v5              v5.x.x
github.com/google/uuid                    v1.x.x
github.com/kelseyhightower/envconfig       v1.4.x
github.com/redis/go-redis/v9              v9.x.x
github.com/minio/minio-go/v7              v7.x.x
golang.org/x/crypto                        latest
github.com/stretchr/testify               v1.x.x
github.com/testcontainers/testcontainers-go v0.x.x
```

### Directory Skeleton
Create the exact Go package structure. Every directory must have at least a `.gitkeep` or a minimal stub file:

```
dogfood/
├── cmd/
│   └── api/
│       └── main.go                      ← package main, empty main() with TODO comment
├── internal/
│   ├── auth/
│   │   ├── domain/user.go               ← package domain — empty stub with TODO
│   │   ├── port/in.go                   ← package port — empty stub with TODO
│   │   ├── port/out.go                  ← package port (same package, different file)
│   │   ├── usecase/.gitkeep
│   │   ├── handler/.gitkeep
│   │   └── repository/.gitkeep
│   ├── events/
│   │   ├── domain/event.go              ← stub
│   │   ├── port/in.go                   ← stub
│   │   ├── port/out.go                  ← stub
│   │   ├── usecase/.gitkeep
│   │   ├── handler/.gitkeep
│   │   └── repository/.gitkeep
│   ├── teams/
│   │   ├── domain/team.go               ← stub
│   │   ├── port/in.go                   ← stub
│   │   ├── port/out.go                  ← stub
│   │   ├── usecase/.gitkeep
│   │   ├── handler/.gitkeep
│   │   └── repository/.gitkeep
│   ├── submissions/
│   │   ├── domain/submission.go         ← stub
│   │   ├── port/in.go                   ← stub
│   │   ├── port/out.go                  ← stub
│   │   ├── usecase/.gitkeep
│   │   ├── handler/.gitkeep
│   │   └── repository/.gitkeep
│   ├── judging/
│   │   ├── domain/score.go              ← stub
│   │   ├── port/in.go                   ← stub
│   │   ├── port/out.go                  ← stub
│   │   ├── usecase/.gitkeep
│   │   ├── handler/.gitkeep
│   │   └── repository/.gitkeep
│   ├── voting/
│   │   ├── domain/vote.go               ← stub
│   │   ├── port/in.go                   ← stub
│   │   ├── port/out.go                  ← stub
│   │   ├── usecase/.gitkeep
│   │   ├── handler/.gitkeep
│   │   └── repository/.gitkeep
│   ├── admin/
│   │   ├── domain/audit.go              ← stub
│   │   ├── port/in.go                   ← stub
│   │   ├── port/out.go                  ← stub
│   │   ├── usecase/.gitkeep
│   │   ├── handler/.gitkeep
│   │   └── repository/.gitkeep
│   └── shared/
│       ├── middleware/.gitkeep
│       ├── response/response.go         ← IMPLEMENT NOW (see below)
│       ├── config/config.go             ← IMPLEMENT NOW (see below)
│       ├── db/.gitkeep
│       ├── cache/.gitkeep
│       └── port/storage.go             ← IMPLEMENT NOW (see below)
├── migrations/.gitkeep
├── Makefile
└── docker-compose.yml                   ← placeholder (full impl in F-002)
```

### shared/response/response.go — Implement fully
This is used by every handler. Build the complete version now:

```go
package response

import (
    "encoding/json"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5/middleware"
)

// ApiResponse is the standard envelope for all successful responses.
type ApiResponse[T any] struct {
    Data T       `json:"data"`
    Meta ApiMeta `json:"meta"`
}

// ApiMeta contains request metadata included in every response.
type ApiMeta struct {
    RequestID  string `json:"requestId"`
    Timestamp  string `json:"timestamp"`
    Page       *int   `json:"page,omitempty"`
    PageSize   *int   `json:"pageSize,omitempty"`
    TotalCount *int   `json:"totalCount,omitempty"`
    TotalPages *int   `json:"totalPages,omitempty"`
}

// ApiError is the standard envelope for all error responses.
type ApiErrorResponse struct {
    Error ApiError `json:"error"`
    Meta  ApiMeta  `json:"meta"`
}

type ApiError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

func newMeta(r *http.Request) ApiMeta {
    reqID := middleware.GetReqID(r.Context())
    return ApiMeta{
        RequestID: reqID,
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    }
}

func JSON[T any](w http.ResponseWriter, r *http.Request, status int, data T) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(ApiResponse[T]{Data: data, Meta: newMeta(r)})
}

func OK[T any](w http.ResponseWriter, r *http.Request, data T) {
    JSON(w, r, http.StatusOK, data)
}

func Created[T any](w http.ResponseWriter, r *http.Request, data T) {
    JSON(w, r, http.StatusCreated, data)
}

func Error(w http.ResponseWriter, r *http.Request, status int, code, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(ApiErrorResponse{
        Error: ApiError{Code: code, Message: message},
        Meta:  newMeta(r),
    })
}

func BadRequest(w http.ResponseWriter, r *http.Request, code, message string) {
    Error(w, r, http.StatusBadRequest, code, message)
}

func Unauthorized(w http.ResponseWriter, r *http.Request, message string) {
    Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func Forbidden(w http.ResponseWriter, r *http.Request, message string) {
    Error(w, r, http.StatusForbidden, "FORBIDDEN", message)
}

func NotFound(w http.ResponseWriter, r *http.Request, message string) {
    Error(w, r, http.StatusNotFound, "NOT_FOUND", message)
}

func InternalError(w http.ResponseWriter, r *http.Request) {
    Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
}
```

### shared/config/config.go — Implement fully

```go
package config

import (
    "github.com/kelseyhightower/envconfig"
)

type Config struct {
    // Server
    Port int    `envconfig:"PORT" default:"8080"`
    Env  string `envconfig:"APP_ENV" default:"development"`

    // Database
    DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`

    // Redis
    RedisURL string `envconfig:"REDIS_URL" required:"true"`

    // JWT
    JWTSecret            string `envconfig:"JWT_SECRET" required:"true"`
    JWTAccessTTLMinutes  int    `envconfig:"JWT_ACCESS_TTL_MINUTES" default:"15"`
    JWTRefreshTTLDays    int    `envconfig:"JWT_REFRESH_TTL_DAYS" default:"7"`

    // MinIO
    MinioEndpoint  string `envconfig:"MINIO_ENDPOINT" required:"true"`
    MinioAccessKey string `envconfig:"MINIO_ACCESS_KEY" required:"true"`
    MinioSecretKey string `envconfig:"MINIO_SECRET_KEY" required:"true"`
    MinioUseSSL    bool   `envconfig:"MINIO_USE_SSL" default:"false"`

    // IP hashing
    IPHashSalt string `envconfig:"IP_HASH_SALT" required:"true"`
}

func Load() (*Config, error) {
    var cfg Config
    if err := envconfig.Process("", &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

### shared/port/storage.go — Implement fully

```go
package port

import (
    "context"
    "io"
)

// FileStorage is the outbound port for object storage.
// Implemented by MinIO adapter. Shared across submissions and admin modules.
type FileStorage interface {
    Upload(ctx context.Context, bucket, objectName, contentType string, size int64, reader io.Reader) (string, error)
    PresignedURL(ctx context.Context, bucket, objectName string) (string, error)
    Delete(ctx context.Context, bucket, objectName string) error
}
```

### Makefile
```makefile
.PHONY: build test vet run migrate docker

build:
	go build ./...

test:
	go test ./... -race -count=1

vet:
	go vet ./...

run:
	go run ./cmd/api/...

migrate:
	migrate -path=./migrations -database=$(DATABASE_URL) up

docker:
	docker compose up --build
```

### Architecture Enforcement (go-arch-lint)
Create `.go-arch-lint.yml` at project root:
```yaml
version: 2
workdir: .
allow:
  depOnAnyVendor: false
components:
  domain:
    in: internal/{module}/domain
  port:
    in: internal/{module}/port
  usecase:
    in: internal/{module}/usecase
  handler:
    in: internal/{module}/handler
  repository:
    in: internal/{module}/repository
  shared:
    in: internal/shared
  cmd:
    in: cmd

deps:
  handler:
    mayDependOn: [ port, domain, shared ]
  usecase:
    mayDependOn: [ port, domain, shared ]
  repository:
    mayDependOn: [ domain, shared ]
  port:
    mayDependOn: [ domain ]
  domain:
    mayDependOn: []  # zero project imports
  cmd:
    mayDependOn: [ handler, usecase, repository, port, domain, shared ]
```

## Tests Required
- `go build ./...` — zero compile errors (no circular imports)
- `go vet ./...` — zero warnings
- One smoke test in `shared/response/response_test.go`:
  - Verify `ApiResponse[T]` serializes to `{"data":{...},"meta":{"requestId":"...","timestamp":"..."}}`
  - Verify `ApiErrorResponse` serializes to `{"error":{"code":"...","message":"..."},"meta":{...}}`

## Definition of Done
- [ ] `go build ./...` → 0 errors
- [ ] `go vet ./...` → 0 warnings
- [ ] All 7 module directories exist with `domain/`, `port/`, `usecase/`, `handler/`, `repository/`
- [ ] `shared/response/response.go` compiles and response_test.go passes
- [ ] `shared/config/config.go` compiles
- [ ] `shared/port/storage.go` compiles
- [ ] `.go-arch-lint.yml` exists
- [ ] `Makefile` exists with `build`, `test`, `vet`, `run`, `migrate`, `docker` targets
