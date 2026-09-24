# TECH STACK
# Dogfood Hackathon Platform

> **Updated**: September 20, 2026 — backend reverted to Go 1.23 (from Java 21)
> **Philosophy**: Production-grade, robust, correct. Every choice justified.
> **Do not change during the 72-hour build without team agreement.**

---

## Stack at a Glance

```
Backend API    Go 1.23 + Chi v5            — HTTP API, business logic, goroutines
Frontend       Next.js 14 (TypeScript)     — Server-side rendered UI (unchanged)
Database       PostgreSQL 16               — Primary data store (unchanged)
Cache          Redis 7                     — Rate limiting, JWT blacklist, signaling (unchanged)
File Storage   MinIO (latest)              — S3-compatible object storage (unchanged)
Proxy          Nginx (alpine)              — Reverse proxy (unchanged)

Docker Compose services: postgres | redis | minio | api | web | nginx
```

Everything except the backend runtime is unchanged. The data model (19 tables), invariants
(I1–I19), state machines, API design (36 endpoints), and Docker topology are 100% preserved.

---

## Why Go (and Why We Reverted from Java)

| Criterion | Go 1.23 | Java 21 + Spring Boot |
|-----------|---------|----------------------|
| Binary size | ~15MB (static binary, scratch) | ~350MB (JRE + JAR) |
| Startup time | <10ms | 2–4s |
| Concurrency | Goroutines (native, lightweight) | Virtual Threads (good, not native) |
| Memory per request | ~4KB goroutine stack | ~1MB thread stack |
| Explicit dependencies | ✅ `go.mod` — no hidden magic | ⚠️ Spring auto-config magic |
| Compile-time safety | ✅ Strong, fast compiler | ✅ Strong, slow compiler |
| Build speed | ✅ Seconds | ⚠️ Minutes (Maven + JVM warmup) |
| DI / IoC | Manual constructor injection | Spring DI framework |
| Learning curve (new) | ✅ Smaller standard library | ⚠️ Large Spring ecosystem |

**Verdict:** Go is a better fit for this project. The data model, invariants, and API are
all pre-designed. Implementation is translation work — Go's explicit style makes it
easier to verify that each invariant is actually enforced in the right place.

---

## Backend: Go 1.23 + Chi v5

### HTTP Router: Chi v5

Chi is the most idiomatic Go router for clean/hexagonal architecture. Every middleware
is just a `func(http.Handler) http.Handler` — pure Go, no magic, fully testable.

```go
// cmd/api/main.go
r := chi.NewRouter()

// Global middleware
r.Use(middleware.RequestID)
r.Use(middleware.RealIP)
r.Use(shared.LoggingMiddleware)
r.Use(shared.RecoverMiddleware)

// Auth routes (public)
r.Route("/api/v1/auth", func(r chi.Router) {
    r.Post("/register", authHandler.Register)
    r.Post("/login",    authHandler.Login)
    r.Post("/refresh",  authHandler.Refresh)
    r.Post("/logout",   authHandler.Logout)
})

// Protected routes
r.Route("/api/v1", func(r chi.Router) {
    r.Use(shared.JWTMiddleware(cfg.JWTSecret))   // validates Bearer token
    r.Use(shared.AuditMiddleware(auditRepo))      // writes audit log for mutations

    r.Route("/events", func(r chi.Router) {
        r.Get("/",         eventHandler.List)
        r.Post("/",        eventHandler.Create)
        r.Get("/{slug}",   eventHandler.Get)
        r.Patch("/{slug}", eventHandler.Update)
        // ...
    })
})
```

**Why Chi over Gin:**
- Chi uses `net/http` stdlib types throughout — no framework lock-in, no `gin.Context` wrapper
- Middleware is composable: `r.Use(...)` on any sub-router
- Pattern matching is clear and explicit
- Easier to unit-test handlers with `httptest.NewRecorder()`
- The hexagonal architecture we've designed maps directly to Chi's structure

---

### Database: pgx v5 + sqlx

**pgx** is the native, high-performance PostgreSQL driver for Go (26k+ GitHub stars).
**sqlx** adds named parameter queries and struct scanning on top of `database/sql`.

```go
// internal/shared/db/postgres.go
func NewPool(ctx context.Context, connStr string) (*sqlx.DB, error) {
    db, err := sqlx.ConnectContext(ctx, "pgx", connStr)
    if err != nil {
        return nil, fmt.Errorf("connect postgres: %w", err)
    }
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(30 * time.Minute)
    return db, nil
}
```

**Named queries (sqlx) — no magic, just SQL:**
```go
// internal/auth/repository/postgres.go
const insertUser = `
    INSERT INTO users (id, email, password_hash, display_name, created_at)
    VALUES (:id, :email, :password_hash, :display_name, :created_at)
`

func (r *UserRepository) Save(ctx context.Context, u *domain.User) error {
    _, err := r.db.NamedExecContext(ctx, insertUser, u)
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            return port.ErrDuplicateEmail   // I1: unique email constraint
        }
        return fmt.Errorf("save user: %w", err)
    }
    return nil
}
```

**Z-score normalization query (raw SQL, full PostgreSQL):**
```go
const normalizationSQL = `
    WITH judge_stats AS (
        SELECT judge_id,
               AVG(raw_score)    AS mu,
               STDDEV(raw_score) AS sigma
        FROM scores
        WHERE submission_id IN (
            SELECT id FROM submissions WHERE event_id = $1
        )
        GROUP BY judge_id
    )
    UPDATE scores s
    SET normalized_score = CASE
          WHEN js.sigma = 0 THEN 0
          ELSE (s.raw_score - js.mu) / js.sigma
        END
    FROM judge_stats js
    WHERE s.judge_id = js.judge_id
`
```

**Why pgx + sqlx over GORM:**
- Every query is visible, explicit, and matches the data model doc exactly
- No hidden N+1 queries (GORM's lazy loading is a footgun)
- pgx supports PostgreSQL-specific features: LISTEN/NOTIFY, COPY, CTEs
- Explicit error handling: catch `pgconn.PgError` for constraint violations → domain errors
- Struct scanning is safe and fast with sqlx's named queries

---

### Database Migrations: golang-migrate v4

Same SQL migration files as designed in `docs/data-model.md`. golang-migrate reads them
from `migrations/` directory and runs them at startup.

```go
// cmd/api/main.go
func runMigrations(connStr string) error {
    m, err := migrate.New(
        "file://migrations",
        connStr,
    )
    if err != nil {
        return fmt.Errorf("create migrator: %w", err)
    }
    if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
        return fmt.Errorf("run migrations: %w", err)
    }
    return nil
}
```

File naming: `001_create_users.up.sql`, `001_create_users.down.sql`, etc.

---

### JWT: golang-jwt/jwt v5

```go
// internal/shared/auth/jwt.go
type Claims struct {
    UserID  uuid.UUID `json:"user_id"`
    IsAdmin bool      `json:"is_admin"`
    jwt.RegisteredClaims
}

func IssueToken(userID uuid.UUID, isAdmin bool, secret []byte, ttl time.Duration) (string, error) {
    claims := Claims{
        UserID:  userID,
        IsAdmin: isAdmin,
        RegisteredClaims: jwt.RegisteredClaims{
            ID:        uuid.NewString(),          // jti — for blacklisting on logout
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(secret)
}

func ParseToken(tokenStr string, secret []byte) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
        if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
        }
        return secret, nil
    })
    // ...
}
```

**JWT spec (unchanged):**
- HS256, signed with `JWT_SECRET` env var
- Access token: 24hr expiry, claims: `user_id`, `jti`, `is_admin`, `exp`
- Refresh token: 7 days, stored SHA-256 hashed in `refresh_tokens` table
- Redis blacklist: `SET revoked:{jti} 1 EX {remaining_ttl_seconds}`

---

### Password Hashing: golang.org/x/crypto bcrypt

```go
// internal/shared/auth/password.go
const BcryptCost = 12  // ~300ms on modern hardware — same security as before

func HashPassword(plain string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(plain), BcryptCost)
    return string(hash), err
}

func VerifyPassword(hash, plain string) bool {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
```

---

### Redis: go-redis/redis v9

```go
// internal/shared/cache/redis.go
type RedisCache struct {
    client *redis.Client
}

// JWT blacklist — O(1) lookup on every authenticated request
func (c *RedisCache) BlacklistJTI(ctx context.Context, jti string, ttl time.Duration) error {
    return c.client.Set(ctx, "revoked:"+jti, "1", ttl).Err()
}

func (c *RedisCache) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
    result, err := c.client.Exists(ctx, "revoked:"+jti).Result()
    return result > 0, err
}

// Sliding window rate limiting via Lua script (atomic)
var rateLimitScript = redis.NewScript(`
    local key = KEYS[1]
    local now = tonumber(ARGV[1])
    local window = tonumber(ARGV[2])
    local limit = tonumber(ARGV[3])
    redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
    local count = redis.call('ZCARD', key)
    if count < limit then
        redis.call('ZADD', key, now, now .. ':' .. math.random())
        redis.call('EXPIRE', key, math.ceil(window / 1000))
        return 0
    end
    return 1
`)

func (c *RedisCache) IsRateLimited(ctx context.Context, key string, windowMs, limit int) (bool, error) {
    result, err := rateLimitScript.Run(ctx, c.client,
        []string{key},
        time.Now().UnixMilli(), windowMs, limit,
    ).Int()
    return result == 1, err
}
```

---

### File Storage: minio-go v7 + FileStorage Interface

```go
// internal/shared/port/storage.go (outbound port — interface)
type FileStorage interface {
    Upload(ctx context.Context, bucket, key string, data io.Reader, size int64, mimeType string) (string, error)
    PresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
    Delete(ctx context.Context, bucket, key string) error
}

// internal/shared/storage/minio.go (outbound adapter — implements FileStorage)
type MinioStorage struct {
    client *minio.Client
}

func (s *MinioStorage) Upload(ctx context.Context, bucket, key string, data io.Reader, size int64, mimeType string) (string, error) {
    _, err := s.client.PutObject(ctx, bucket, key, data, size, minio.PutObjectOptions{
        ContentType: mimeType,
    })
    if err != nil {
        return "", fmt.Errorf("minio upload: %w", err)
    }
    return fmt.Sprintf("/%s/%s", bucket, key), nil
}
```

---

### Config: kelseyhightower/envconfig

```go
// internal/shared/config/config.go
type Config struct {
    // Server
    Port int    `envconfig:"PORT" default:"8080"`

    // Database
    DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`

    // Redis
    RedisURL string `envconfig:"REDIS_URL" required:"true"`

    // MinIO
    MinioEndpoint  string `envconfig:"MINIO_ENDPOINT"   required:"true"`
    MinioAccessKey string `envconfig:"MINIO_ACCESS_KEY" required:"true"`
    MinioSecretKey string `envconfig:"MINIO_SECRET_KEY" required:"true"`

    // JWT
    JWTSecret          string        `envconfig:"JWT_SECRET"             required:"true"`
    JWTAccessTokenTTL  time.Duration `envconfig:"JWT_ACCESS_TTL"         default:"24h"`
    JWTRefreshTokenTTL time.Duration `envconfig:"JWT_REFRESH_TTL"        default:"168h"`

    // App
    Environment string `envconfig:"ENV" default:"development"`
}

func Load() (*Config, error) {
    var cfg Config
    if err := envconfig.Process("", &cfg); err != nil {
        return nil, fmt.Errorf("load config: %w", err)
    }
    return &cfg, nil
}
```

---

### Structured Logging: log/slog (Go 1.21+ stdlib)

Zero dependency. Built into Go 1.21+.

```go
// internal/shared/middleware/logging.go
func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
        next.ServeHTTP(ww, r)
        slog.InfoContext(r.Context(), "request",
            slog.String("method",     r.Method),
            slog.String("path",       r.URL.Path),
            slog.Int("status",        ww.Status()),
            slog.Duration("duration", time.Since(start)),
            slog.String("request_id", middleware.GetReqID(r.Context())),
        )
    })
}
```

Output: `{"time":"2026-09-20T15:00:00Z","level":"INFO","msg":"request","method":"POST","path":"/api/v1/auth/register","status":201,"duration":"14ms"}`

---

### Background Jobs: Goroutines

No framework needed. Goroutines are native in Go — just `go func()`.

```go
// internal/judging/usecase/normalize.go
func (s *NormalizationService) TriggerNormalization(ctx context.Context, eventID uuid.UUID) error {
    // 1. Check all assignments complete (I13)
    if err := s.checkAllComplete(ctx, eventID); err != nil {
        return err
    }
    // 2. Mark as in_progress
    if err := s.eventRepo.SetNormalizationStatus(ctx, eventID, "in_progress"); err != nil {
        return err
    }
    // 3. Run in background goroutine — returns 202 immediately
    go func() {
        bgCtx := context.Background()  // detached from request context
        if err := s.runNormalization(bgCtx, eventID); err != nil {
            slog.Error("normalization failed", "event_id", eventID, "error", err)
            _ = s.eventRepo.SetNormalizationStatus(bgCtx, eventID, "failed")
            return
        }
        _ = s.eventRepo.SetNormalizationStatus(bgCtx, eventID, "normalized")
        _ = s.redisCache.Publish(bgCtx, "events:normalization", eventID.String())
    }()
    return nil  // HTTP handler returns 202
}
```

---

### Testing: testify + testcontainers-go + httptest

```go
// internal/auth/handler/register_test.go
func TestRegisterHandler(t *testing.T) {
    // Real PostgreSQL via testcontainers
    ctx := context.Background()
    pg, _ := testcontainers.RunContainer(ctx, postgres.RunContainer,
        postgres.WithDatabase("testdb"),
        postgres.WithImage("postgres:16-alpine"),
    )
    defer pg.Terminate(ctx)

    // Wire up handler
    db := setupTestDB(t, pg)
    handler := NewAuthHandler(NewRegisterService(NewUserRepository(db), bcrypt.NewHasher()))

    t.Run("returns 201 with correct shape", func(t *testing.T) {
        body := `{"email":"alice@test.com","password":"Secure123!","displayName":"Alice"}`
        req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        w := httptest.NewRecorder()

        handler.Register(w, req)

        assert.Equal(t, http.StatusCreated, w.Code)
        var resp response.ApiResponse[RegisterResponse]
        assert.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
        assert.Equal(t, "alice@test.com", resp.Data.Email)
    })

    t.Run("cannotRegisterWithDuplicateEmail", func(t *testing.T) {
        // ... register once, try again → assert 409 DUPLICATE_RESOURCE
    })
}
```

- **Unit tests**: `testing` + `testify/assert` for pure logic (state machines, normalization math)
- **Integration tests**: `testcontainers-go` spins up real PostgreSQL 16 — no mocking DB
- **HTTP tests**: `httptest.NewRecorder()` tests the full handler chain
- **No SQLite or in-memory DB** — always test against real PostgreSQL

---

## go.mod Dependencies

```
module github.com/dogfood/hackathon-platform

go 1.23

require (
    github.com/go-chi/chi/v5          v5.1.0    // HTTP router
    github.com/jackc/pgx/v5           v5.7.0    // PostgreSQL driver
    github.com/jmoiron/sqlx           v1.4.0    // Named queries + struct scanning
    github.com/golang-migrate/migrate/v4 v4.18.0 // DB migrations
    github.com/golang-jwt/jwt/v5      v5.2.1    // JWT sign/verify
    github.com/go-redis/redis/v9      v9.7.0    // Redis client
    github.com/minio/minio-go/v7      v7.0.78   // MinIO SDK
    github.com/kelseyhightower/envconfig v1.4.0  // Env config
    github.com/google/uuid            v1.6.0    // UUID generation
    github.com/stretchr/testify       v1.9.0    // Test assertions
    github.com/testcontainers/testcontainers-go v0.35.0 // Real DB in tests
    golang.org/x/crypto               v0.28.0   // bcrypt
)
```

---

## What This Stack Does NOT Include

| Excluded | Why |
|----------|-----|
| GORM or any ORM | pgx + sqlx gives full SQL control; ORM hides invariant violations |
| Gin | Chi is cleaner for hexagonal architecture; no `gin.Context` wrapper |
| Kafka / RabbitMQ | Redis pub/sub sufficient for normalization signaling |
| GraphQL | REST only; acceptance suite expects REST endpoints |
| External auth (OAuth) | Custom JWT only per MASTER-CONTEXT.md |
| Wire / dig (DI frameworks) | Manual constructor injection — explicit, readable, fast |
| GraalVM / Native Go plugins | Standard `go build` — 10-second binary |

---

## Docker Image Sizes (estimated)

```
postgres:16-alpine       ~250MB
redis:7-alpine            ~35MB
minio/minio              ~100MB
api (scratch binary)      ~15MB   ← Go static binary, tiny
web (next.js standalone) ~200MB
nginx:alpine              ~25MB

Total stack:             ~625MB   (vs ~960MB with Java — 35% smaller)
```

---

*Next: ARCHITECTURE.md — Go package structure (internal/), hexagonal adapter map, request lifecycle, invariant enforcement*
