---
id: A-003
title: Token Refresh + Logout
epic: auth
owner: Sujay (backend)
status: "[ ] not-started"
branch: story/A-003-refresh-logout
blocks: T-001
blocked-by: A-002
---

# A-003 · Token Refresh + Logout

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules, JWT middleware, Redis cache
- `MASTER-CONTEXT.md` — JWT blacklist: `revoked:{jti}` key in Redis with TTL matching token expiry
- `docs/api-design.md §Auth` — exact shapes for `POST /auth/refresh` and `POST /auth/logout`
- `docs/data-model.md §refresh_tokens` — `revoked_at` column, `token_hash` for lookup

## What We're Building

Two endpoints:

1. **Token Refresh** — Accepts a refresh token, verifies it, issues a NEW access token (and a NEW refresh token — refresh token rotation), revokes the old refresh token in DB.

2. **Logout** — Accepts the current access token (via Bearer header), blacklists its JTI in Redis with TTL = remaining time until expiry. Also revokes the specific refresh token if provided in the request body.

**Endpoints:**
- `POST /api/v1/auth/refresh` — public (no JWT middleware, takes refresh token in body)
- `POST /api/v1/auth/logout` — authenticated (requires valid JWT, blacklists it)

### POST /auth/refresh

**Request:**
```json
{ "refreshToken": "eyJ..." }
```
**Success (200):**
```json
{
  "data": { "accessToken": "eyJ...", "refreshToken": "eyJ..." },
  "meta": { "requestId": "...", "timestamp": "..." }
}
```
**Errors:**
| Condition | HTTP | Code |
|-----------|------|------|
| Token invalid/expired | 401 | `INVALID_REFRESH_TOKEN` |
| Token already revoked | 401 | `TOKEN_REVOKED` |

### POST /auth/logout

**Request:** Bearer token in `Authorization` header  
**Success (200):**
```json
{ "data": { "loggedOut": true }, "meta": {...} }
```
**Note:** Logout NEVER fails with an error (idempotent). If the token is already expired or revoked, return 200 anyway.

## Files to Create / Modify

### internal/auth/port/in.go (add)
```go
type RefreshUseCase interface {
    Refresh(ctx context.Context, cmd RefreshCommand) (*TokenPair, error)
}

type LogoutUseCase interface {
    Logout(ctx context.Context, jti string, expiresAt time.Time) error
}

type RefreshCommand struct {
    RawRefreshToken string
}

type TokenPair struct {
    AccessToken  string `json:"accessToken"`
    RefreshToken string `json:"refreshToken"`
}
```

### internal/auth/port/out.go (add)
```go
// CacheStore is the Redis outbound port for the auth module.
// Used to blacklist revoked JTIs.
type CacheStore interface {
    Set(ctx context.Context, key string, value string, ttl time.Duration) error
    Exists(ctx context.Context, key string) (bool, error)
}
```

### internal/auth/usecase/refresh.go
```go
package usecase

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/dogfood/platform/internal/auth/domain"
    "github.com/dogfood/platform/internal/auth/port"
    "github.com/dogfood/platform/internal/shared/response"
)

type RefreshService struct {
    users   port.UserRepository
    refresh port.RefreshTokenRepository
    tokens  port.TokenIssuer
}

func (s *RefreshService) Refresh(ctx context.Context, cmd port.RefreshCommand) (*port.TokenPair, error) {
    tokenHash := domain.HashToken(cmd.RawRefreshToken)

    // 1. Find the refresh token by its hash
    rt, err := s.refresh.FindByHash(ctx, tokenHash)
    if err != nil {
        return nil, fmt.Errorf("%w: invalid refresh token", response.ErrUnauthorized)
    }

    // 2. Check if expired or revoked
    if rt.IsExpired() {
        return nil, fmt.Errorf("%w: refresh token has been revoked or expired", response.ErrUnauthorized)
    }

    // 3. Load user
    user, err := s.users.FindByID(ctx, rt.UserID)
    if err != nil || !user.IsActive {
        return nil, fmt.Errorf("%w: user not found or disabled", response.ErrUnauthorized)
    }

    // 4. Revoke the old refresh token (rotation)
    if err := s.refresh.Revoke(ctx, rt.ID); err != nil {
        return nil, fmt.Errorf("revoke old refresh token: %w", err)
    }

    // 5. Issue new tokens
    newAccess, err := s.tokens.IssueAccessToken(user.ID.String(), user.Email, user.IsAdmin)
    if err != nil {
        return nil, fmt.Errorf("issue access token: %w", err)
    }
    newRaw, err := s.tokens.IssueRefreshToken(user.ID.String())
    if err != nil {
        return nil, fmt.Errorf("issue refresh token: %w", err)
    }

    // 6. Store new refresh token
    newRT := &domain.RefreshToken{
        ID:        uuid.New(),
        UserID:    user.ID,
        TokenHash: domain.HashToken(newRaw),
        ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
        CreatedAt: time.Now().UTC(),
    }
    if err := s.refresh.Save(ctx, newRT); err != nil {
        return nil, fmt.Errorf("save new refresh token: %w", err)
    }

    return &port.TokenPair{AccessToken: newAccess, RefreshToken: newRaw}, nil
}
```

### internal/auth/usecase/logout.go
```go
package usecase

import (
    "context"
    "fmt"
    "time"

    "github.com/dogfood/platform/internal/auth/port"
)

type LogoutService struct {
    cache port.CacheStore
}

func (s *LogoutService) Logout(ctx context.Context, jti string, expiresAt time.Time) error {
    ttl := time.Until(expiresAt)
    if ttl <= 0 {
        return nil // Token already expired — blacklisting it is a no-op
    }
    // Write to Redis: SET revoked:{jti} 1 EX <ttl_seconds>
    key := fmt.Sprintf("revoked:%s", jti)
    return s.cache.Set(ctx, key, "1", ttl)
}
```

### internal/auth/handler/handler.go (add Refresh + Logout)
```go
func (h *AuthHandler) Routes() chi.Router {
    r := chi.NewRouter()
    r.Post("/register", h.Register)
    r.Post("/login", h.Login)
    r.Post("/refresh", h.Refresh)   // ← public
    r.Group(func(r chi.Router) {
        r.Use(middleware.JWTMiddleware(h.jwtSecret, h.cache))
        r.Post("/logout", h.Logout) // ← authenticated
    })
    return r
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
    var req struct {
        RefreshToken string `json:"refreshToken"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
        response.BadRequest(w, r, "VALIDATION_ERROR", "refreshToken is required")
        return
    }
    pair, err := h.refresh.Refresh(r.Context(), port.RefreshCommand{RawRefreshToken: req.RefreshToken})
    if err != nil {
        response.HandleDomainError(w, r, err)
        return
    }
    response.OK(w, r, pair)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
    // JTI and expiry are extracted from the validated JWT in context
    jti := middleware.GetJTI(r.Context())
    exp := middleware.GetTokenExpiry(r.Context())
    // Errors are intentionally swallowed — logout is always 200
    _ = h.logout.Logout(r.Context(), jti, exp)
    response.OK(w, r, map[string]bool{"loggedOut": true})
}
```

### internal/shared/cache/redis.go — Redis adapter
```go
package cache

import (
    "context"
    "time"

    "github.com/redis/go-redis/v9"
)

type RedisCache struct {
    client *redis.Client
}

func NewRedisCache(redisURL string) (*RedisCache, error) {
    opts, err := redis.ParseURL(redisURL)
    if err != nil {
        return nil, err
    }
    client := redis.NewClient(opts)
    return &RedisCache{client: client}, nil
}

func (c *RedisCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
    return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
    n, err := c.client.Exists(ctx, key).Result()
    return n > 0, err
}
```

## Tests Required

### Refresh use case tests
- `TestRefreshService_ValidToken_ReturnsNewTokenPair` — old token revoked, new tokens issued
- `TestRefreshService_ExpiredToken_Returns401`
- `TestRefreshService_RevokedToken_Returns401`
- `TestRefreshService_TokenRotation_OldTokenNoLongerWorks` — after refresh, using old token → 401

### Logout use case tests
- `TestLogoutService_ValidJTI_BlacklistsInRedis`
- `TestLogoutService_AlreadyExpiredToken_NoOp_Returns200` — no error thrown

### Handler tests
- `TestRefreshHandler_ValidToken_Returns200_WithNewPair`
- `TestRefreshHandler_MissingToken_Returns400`
- `TestRefreshHandler_ExpiredToken_Returns401_INVALID_REFRESH_TOKEN`
- `TestLogoutHandler_ValidJWT_Returns200_LoggedOut`

### Integration test
- `POST /auth/login` → get tokens
- `POST /auth/refresh` → verify new access token works, old refresh token rejected
- `POST /auth/logout` → blacklist JTI
- `GET /api/v1/me` with old access token → 401 `TOKEN_REVOKED`

## Definition of Done
- [ ] `go test ./internal/auth/...` → 100% green
- [ ] `POST /auth/refresh` with valid refresh token → 200 new token pair
- [ ] Old refresh token after rotation → 401 `TOKEN_REVOKED`
- [ ] `POST /auth/logout` → 200 `{"loggedOut":true}`
- [ ] After logout, access token → 401 (blacklisted in Redis)
- [ ] Logout with already-expired token → 200 (idempotent, no error)
- [ ] Redis key `revoked:{jti}` exists with correct TTL after logout
