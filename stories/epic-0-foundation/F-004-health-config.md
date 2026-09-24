---
id: F-004
title: Health Endpoint + Chi Router + Structured Logging Setup
epic: foundation
owner: both
status: "[ ] not-started"
branch: story/F-004-health-config
blocks: A-001
blocked-by: F-003
---

# F-004 · Health Endpoint + Chi Router + Structured Logging Setup

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — middleware chain, request lifecycle, error envelope
- `MASTER-CONTEXT.md` — confirmed stack: Chi v5, slog (stdlib), golang-jwt v5
- `docs/architecture.md §3.1` — the 8-step request lifecycle (request ID → JWT → role → rate limit → handler → use case → response)
- `docs/api-design.md` — standard response envelope shape `{data, meta}` and error shape `{error, meta}`

## What to Build

### cmd/api/main.go — Fully wired server
The entry point that wires everything together and starts the Chi server:

```go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/go-chi/chi/v5"
    chimiddleware "github.com/go-chi/chi/v5/middleware"

    "github.com/dogfood/platform/internal/shared/config"
    "github.com/dogfood/platform/internal/shared/db"
    "github.com/dogfood/platform/internal/shared/middleware"
)

func main() {
    // Structured logging (slog — stdlib Go 1.21+)
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
    slog.SetDefault(logger)

    // Load config from env
    cfg, err := config.Load()
    if err != nil {
        slog.Error("failed to load config", "error", err)
        os.Exit(1)
    }

    // Database
    ctx := context.Background()
    database, err := db.NewPool(ctx, cfg.DatabaseURL)
    if err != nil {
        slog.Error("failed to connect to database", "error", err)
        os.Exit(1)
    }
    defer database.Close()

    // Run migrations
    if err := db.RunMigrations(database.DB, "./migrations"); err != nil {
        slog.Error("failed to run migrations", "error", err)
        os.Exit(1)
    }
    slog.Info("migrations applied successfully")

    // Chi router
    r := chi.NewRouter()

    // Global middleware (order matters — see architecture.md §3.1)
    r.Use(chimiddleware.RequestID)           // Step 2: inject X-Request-ID
    r.Use(middleware.StructuredLogger(logger)) // Step 2: slog per-request logging
    r.Use(chimiddleware.Recoverer)           // Panic recovery → 500
    r.Use(chimiddleware.RealIP)              // Trust X-Forwarded-For

    // Health endpoint (public, no auth)
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"up","version":"1.0.0"}`))
    })

    // TODO: mount module routes here as stories are completed
    // r.Mount("/api/v1/auth", authHandler.Routes())
    // r.Mount("/api/v1/events", eventHandler.Routes())

    // Graceful shutdown
    srv := &http.Server{
        Addr:         fmt.Sprintf(":%d", cfg.Port),
        Handler:      r,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        slog.Info("server starting", "port", cfg.Port)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            slog.Error("server error", "error", err)
            os.Exit(1)
        }
    }()

    <-quit
    slog.Info("server shutting down gracefully")

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := srv.Shutdown(shutdownCtx); err != nil {
        slog.Error("server forced shutdown", "error", err)
    }
    slog.Info("server exited")
}
```

### internal/shared/middleware/logging.go — Structured request logger

```go
package middleware

import (
    "log/slog"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5/middleware"
)

func StructuredLogger(logger *slog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
            start := time.Now()

            defer func() {
                logger.InfoContext(r.Context(), "http request",
                    "method", r.Method,
                    "path", r.URL.Path,
                    "status", ww.Status(),
                    "bytes", ww.BytesWritten(),
                    "duration_ms", time.Since(start).Milliseconds(),
                    "request_id", middleware.GetReqID(r.Context()),
                    "remote_addr", r.RemoteAddr,
                )
            }()

            next.ServeHTTP(ww, r)
        })
    }
}
```

### internal/shared/middleware/auth.go — JWT validation middleware
Implements step 3 of the request lifecycle (see architecture.md §3.1):

```go
package middleware

import (
    "context"
    "log/slog"
    "net/http"
    "strings"

    "github.com/golang-jwt/jwt/v5"
    "github.com/dogfood/platform/internal/shared/response"
)

type contextKey string

const (
    ContextKeyUserID    contextKey = "userID"
    ContextKeyUserEmail contextKey = "userEmail"
    ContextKeyIsAdmin   contextKey = "isAdmin"
)

type Claims struct {
    UserID  string `json:"sub"`
    Email   string `json:"email"`
    IsAdmin bool   `json:"is_admin"`
    JTI     string `json:"jti"`
    jwt.RegisteredClaims
}

// JWTMiddleware validates Bearer tokens. Routes that do NOT require auth should
// be mounted OUTSIDE this middleware.
func JWTMiddleware(jwtSecret string, cache CacheStore) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authHeader := r.Header.Get("Authorization")
            if !strings.HasPrefix(authHeader, "Bearer ") {
                response.Unauthorized(w, r, "missing or invalid authorization header")
                return
            }
            tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

            claims := &Claims{}
            token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
                if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
                    return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
                }
                return []byte(jwtSecret), nil
            })
            if err != nil || !token.Valid {
                response.Unauthorized(w, r, "invalid or expired token")
                return
            }

            // Check Redis blacklist: EXISTS revoked:{jti}
            blacklisted, err := cache.Exists(r.Context(), "revoked:"+claims.JTI)
            if err != nil {
                slog.ErrorContext(r.Context(), "cache check failed", "error", err)
                response.InternalError(w, r)
                return
            }
            if blacklisted {
                response.Unauthorized(w, r, "token has been revoked")
                return
            }

            // Inject user info into context
            ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
            ctx = context.WithValue(ctx, ContextKeyUserEmail, claims.Email)
            ctx = context.WithValue(ctx, ContextKeyIsAdmin, claims.IsAdmin)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// GetUserID extracts the authenticated user ID from context.
// Panics if called on an unauthenticated route (programmer error).
func GetUserID(ctx context.Context) string {
    v, _ := ctx.Value(ContextKeyUserID).(string)
    return v
}

func GetIsAdmin(ctx context.Context) bool {
    v, _ := ctx.Value(ContextKeyIsAdmin).(bool)
    return v
}
```

### internal/shared/middleware/interfaces.go — Middleware dependency interfaces
```go
package middleware

import "context"

// CacheStore is the subset of cache operations the middleware needs.
// Prevents middleware from depending on the full cache package.
type CacheStore interface {
    Exists(ctx context.Context, key string) (bool, error)
}
```

### shared/response/errors.go — Domain error → HTTP mapping
```go
package response

import (
    "errors"
    "net/http"
    "strings"
)

// Domain error sentinels — defined here as a shared contract.
// Each module's domain/ package wraps these with fmt.Errorf("%w: detail", ErrXxx).
var (
    ErrNotFound          = errors.New("not found")
    ErrForbidden         = errors.New("forbidden")
    ErrUnauthorized      = errors.New("unauthorized")
    ErrInvariantViolated = errors.New("invariant violated")
    ErrInvalidTransition = errors.New("invalid state transition")
    ErrDeadlinePassed    = errors.New("deadline has passed")
    ErrRateLimited       = errors.New("rate limited")
)

// HandleDomainError maps domain errors to HTTP responses.
// Every handler uses this — never manually write 422/409 in handlers.
func HandleDomainError(w http.ResponseWriter, r *http.Request, err error) {
    switch {
    case errors.Is(err, ErrNotFound):
        Error(w, r, http.StatusNotFound, "NOT_FOUND", err.Error())
    case errors.Is(err, ErrForbidden):
        Error(w, r, http.StatusForbidden, "FORBIDDEN", err.Error())
    case errors.Is(err, ErrUnauthorized):
        Unauthorized(w, r, err.Error())
    case errors.Is(err, ErrInvalidTransition):
        Error(w, r, http.StatusUnprocessableEntity, "INVALID_STATE_TRANSITION", err.Error())
    case errors.Is(err, ErrDeadlinePassed):
        Error(w, r, http.StatusUnprocessableEntity, "DEADLINE_PASSED", err.Error())
    case errors.Is(err, ErrRateLimited):
        Error(w, r, http.StatusTooManyRequests, "RATE_LIMITED", err.Error())
    case errors.Is(err, ErrInvariantViolated):
        msg := err.Error()
        if strings.Contains(msg, "duplicate") || strings.Contains(msg, "already") ||
            strings.Contains(msg, "unique") || strings.Contains(msg, "exists") {
            Error(w, r, http.StatusConflict, "DUPLICATE_RESOURCE", msg)
        } else {
            Error(w, r, http.StatusUnprocessableEntity, "INVARIANT_VIOLATION", msg)
        }
    default:
        slog.ErrorContext(r.Context(), "unhandled domain error", "error", err)
        InternalError(w, r)
    }
}
```

## Tests Required
- `go build ./cmd/api/...` — zero errors
- `go vet ./...` — zero warnings
- Handler test for `/health`:
  - `GET /health` → 200 `{"status":"up","version":"1.0.0"}`
- Middleware unit tests:
  - `JWTMiddleware` with missing header → 401 `UNAUTHORIZED`
  - `JWTMiddleware` with expired token → 401 `UNAUTHORIZED`
  - `JWTMiddleware` with blacklisted JTI → 401 `UNAUTHORIZED`
  - `JWTMiddleware` with valid token → injects UserID into context
- `HandleDomainError` tests:
  - `ErrNotFound` → 404 `NOT_FOUND`
  - `ErrInvalidTransition` → 422 `INVALID_STATE_TRANSITION`
  - `ErrDeadlinePassed` → 422 `DEADLINE_PASSED`
  - `ErrInvariantViolated` wrapping "already exists" → 409 `DUPLICATE_RESOURCE`
  - `ErrInvariantViolated` (generic) → 422 `INVARIANT_VIOLATION`
  - Unknown error → 500 `INTERNAL_ERROR`

## Definition of Done
- [ ] `go build ./cmd/api/...` → 0 errors
- [ ] `go run ./cmd/api/...` starts server, logs JSON, GET /health → 200
- [ ] `docker compose up api` → `/health` returns `{"status":"up","version":"1.0.0"}`
- [ ] All middleware unit tests pass
- [ ] All `HandleDomainError` tests pass
- [ ] Every log line is JSON (structured via slog.NewJSONHandler)
- [ ] Graceful shutdown works (SIGTERM → server stops within 30s)
