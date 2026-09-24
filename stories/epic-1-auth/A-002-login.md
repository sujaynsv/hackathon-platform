---
id: A-002
title: Login — POST /api/v1/auth/login
epic: auth
owner: Sujay (backend)
status: "[ ] not-started"
branch: story/A-002-login
blocks: A-003, FE-A-001
blocked-by: A-001
---

# A-002 · Login — POST /api/v1/auth/login

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules (§2–§8)
- `MASTER-CONTEXT.md` — JWT config: HS256, access TTL 15min, refresh TTL 7 days
- `docs/api-design.md §Auth` — exact request/response for `POST /api/v1/auth/login`
- `docs/data-model.md §users, §refresh_tokens` — `is_active` flag, `token_hash` column

## What We're Building

Login endpoint. Verifies the user's email and bcrypt password, checks `is_active = true`, issues a new JWT access token (HS256, 15-min TTL) and a refresh token (7-day TTL), stores the refresh token hash in PostgreSQL, and returns both. If the user had a prior refresh token, it is NOT invalidated — multiple devices can be logged in simultaneously (each device has its own refresh token row).

**Endpoint:** `POST /api/v1/auth/login`  
**Auth required:** None (public)

**Request:**
```json
{ "email": "alice@example.com", "password": "Password123!" }
```

**Success response (200 OK):**
```json
{
  "data": {
    "user": { "id": "uuid", "email": "alice@example.com", "displayName": "Alice", "avatarUrl": null, "isAdmin": false, "createdAt": "..." },
    "accessToken": "eyJ...",
    "refreshToken": "eyJ..."
  },
  "meta": { "requestId": "uuid", "timestamp": "..." }
}
```

**Error cases:**
| Condition | HTTP | Code |
|-----------|------|------|
| User not found OR password wrong | 401 | `INVALID_CREDENTIALS` |
| Account deactivated (`is_active = false`) | 403 | `ACCOUNT_DISABLED` |
| Missing fields | 400 | `VALIDATION_ERROR` |

**Security note:** NEVER tell the caller whether the email exists. Always return 401 `INVALID_CREDENTIALS` for both "user not found" and "wrong password" — this prevents user enumeration attacks.

## Files to Create / Modify

### internal/auth/domain/refresh_token.go
```go
package domain

import (
    "crypto/sha256"
    "encoding/hex"
    "time"
    "github.com/google/uuid"
)

type RefreshToken struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    TokenHash string    // SHA-256 of the raw token — never store raw token
    ExpiresAt time.Time
    RevokedAt *time.Time
    CreatedAt time.Time
}

// IsExpired returns true if the token is past its expiry or revoked.
func (rt *RefreshToken) IsExpired() bool {
    return time.Now().UTC().After(rt.ExpiresAt) || rt.RevokedAt != nil
}

// HashToken converts a raw refresh token string to its SHA-256 hex hash for storage.
func HashToken(raw string) string {
    sum := sha256.Sum256([]byte(raw))
    return hex.EncodeToString(sum[:])
}
```

### internal/auth/port/in.go (add LoginUseCase)
```go
// LoginUseCase is the inbound port for user login.
type LoginUseCase interface {
    Login(ctx context.Context, cmd LoginCommand) (*AuthResponse, error)
}

type LoginCommand struct {
    Email    string
    Password string
}
```

### internal/auth/usecase/login.go
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

type LoginService struct {
    users   port.UserRepository
    hasher  port.PasswordHasher
    tokens  port.TokenIssuer
    refresh port.RefreshTokenRepository
    cache   port.CacheStore
}

func NewLoginService(
    users port.UserRepository,
    hasher port.PasswordHasher,
    tokens port.TokenIssuer,
    refresh port.RefreshTokenRepository,
) *LoginService {
    return &LoginService{users: users, hasher: hasher, tokens: tokens, refresh: refresh}
}

func (s *LoginService) Login(ctx context.Context, cmd port.LoginCommand) (*port.AuthResponse, error) {
    // 1. Find user by email — return 401 INVALID_CREDENTIALS for "not found"
    user, err := s.users.FindByEmail(ctx, cmd.Email)
    if err != nil {
        // Use a generic error: never reveal if email exists
        return nil, fmt.Errorf("%w: invalid credentials", response.ErrUnauthorized)
    }

    // 2. Check account is active
    if !user.IsActive {
        return nil, fmt.Errorf("%w: account is disabled", response.ErrForbidden)
    }

    // 3. Verify password (timing-safe bcrypt compare via hasher port)
    if !s.hasher.Verify(user.PasswordHash, cmd.Password) {
        return nil, fmt.Errorf("%w: invalid credentials", response.ErrUnauthorized)
    }

    // 4. Issue tokens
    accessToken, err := s.tokens.IssueAccessToken(user.ID.String(), user.Email, user.IsAdmin)
    if err != nil {
        return nil, fmt.Errorf("issue access token: %w", err)
    }
    rawRefreshToken, err := s.tokens.IssueRefreshToken(user.ID.String())
    if err != nil {
        return nil, fmt.Errorf("issue refresh token: %w", err)
    }

    // 5. Store hashed refresh token
    rt := &domain.RefreshToken{
        ID:        uuid.New(),
        UserID:    user.ID,
        TokenHash: domain.HashToken(rawRefreshToken),
        ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
        CreatedAt: time.Now().UTC(),
    }
    if err := s.refresh.Save(ctx, rt); err != nil {
        return nil, fmt.Errorf("save refresh token: %w", err)
    }

    return &port.AuthResponse{
        User: port.UserDTO{
            ID:          user.ID.String(),
            Email:       user.Email,
            DisplayName: user.DisplayName,
            AvatarURL:   user.AvatarURL,
            IsAdmin:     user.IsAdmin,
            CreatedAt:   user.CreatedAt.Format(time.RFC3339),
        },
        AccessToken:  accessToken,
        RefreshToken: rawRefreshToken,
    }, nil
}
```

### internal/auth/handler/handler.go (add Login)
```go
func (h *AuthHandler) Routes() chi.Router {
    r := chi.NewRouter()
    r.Post("/register", h.Register)
    r.Post("/login", h.Login)     // ← add
    return r
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.BadRequest(w, r, "VALIDATION_ERROR", "invalid request body")
        return
    }
    if req.Email == "" || req.Password == "" {
        response.BadRequest(w, r, "VALIDATION_ERROR", "email and password are required")
        return
    }
    result, err := h.login.Login(r.Context(), port.LoginCommand{
        Email:    req.Email,
        Password: req.Password,
    })
    if err != nil {
        response.HandleDomainError(w, r, err)
        return
    }
    response.OK(w, r, result)
}
```

**Add to shared/response/errors.go** — handle the new auth error codes:
```go
// In HandleDomainError, before the default case:
case errors.Is(err, ErrUnauthorized):
    // For login failures, use a distinct code
    msg := err.Error()
    if strings.Contains(msg, "invalid credentials") {
        Error(w, r, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
    } else {
        Unauthorized(w, r, msg)
    }
case errors.Is(err, ErrForbidden):
    msg := err.Error()
    if strings.Contains(msg, "disabled") {
        Error(w, r, http.StatusForbidden, "ACCOUNT_DISABLED", msg)
    } else {
        Forbidden(w, r, msg)
    }
```

## Adapter Implementations to Create

### internal/shared/auth/bcrypt.go
```go
package auth

import "golang.org/x/crypto/bcrypt"

type BcryptHasher struct{ cost int }

func NewBcryptHasher() *BcryptHasher { return &BcryptHasher{cost: 12} }

func (h *BcryptHasher) Hash(plain string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
    return string(bytes), err
}

func (h *BcryptHasher) Verify(hash, plain string) bool {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
```

### internal/shared/auth/jwt.go
```go
package auth

import (
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

type JWTIssuer struct {
    secret        []byte
    accessTTLMin  int
    refreshTTLDay int
}

func NewJWTIssuer(secret string, accessTTLMin, refreshTTLDay int) *JWTIssuer {
    return &JWTIssuer{
        secret:        []byte(secret),
        accessTTLMin:  accessTTLMin,
        refreshTTLDay: refreshTTLDay,
    }
}

type dogfoodClaims struct {
    UserID  string `json:"sub"`
    Email   string `json:"email"`
    IsAdmin bool   `json:"is_admin"`
    JTI     string `json:"jti"`
    jwt.RegisteredClaims
}

func (j *JWTIssuer) IssueAccessToken(userID, email string, isAdmin bool) (string, error) {
    claims := dogfoodClaims{
        UserID:  userID,
        Email:   email,
        IsAdmin: isAdmin,
        JTI:     uuid.New().String(),
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(j.accessTTLMin) * time.Minute)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signed, err := token.SignedString(j.secret)
    if err != nil {
        return "", fmt.Errorf("sign token: %w", err)
    }
    return signed, nil
}

func (j *JWTIssuer) IssueRefreshToken(userID string) (string, error) {
    claims := dogfoodClaims{
        UserID: userID,
        JTI:    uuid.New().String(),
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(j.refreshTTLDay) * 24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(j.secret)
}
```

## Tests Required

### Use case tests (usecase/login_test.go)
- `TestLoginService_ValidCredentials_ReturnsAuthResponse`
- `TestLoginService_WrongPassword_ReturnsErrUnauthorized` (not ErrNotFound!)
- `TestLoginService_UserNotFound_ReturnsErrUnauthorized` (same 401, not 404 — anti-enumeration)
- `TestLoginService_InactiveUser_ReturnsErrForbidden`

### Handler tests (handler/handler_test.go)
- `TestLoginHandler_ValidCredentials_Returns200`
- `TestLoginHandler_WrongPassword_Returns401WithCode_INVALID_CREDENTIALS`
- `TestLoginHandler_AccountDisabled_Returns403WithCode_ACCOUNT_DISABLED`
- `TestLoginHandler_MissingFields_Returns400`

### JWT adapter tests (shared/auth/jwt_test.go)
- `TestJWTIssuer_IssueAndParseAccessToken` — verify claims round-trip
- `TestJWTIssuer_ExpiredToken_FailsValidation`

### Integration test
- Full `POST /api/v1/auth/login` against testcontainers PostgreSQL
- Insert user → login → verify 200 + parse JWT header

## Definition of Done
- [ ] `go test ./internal/auth/...` → 100% green
- [ ] `POST /api/v1/auth/login` with correct credentials → 200 with tokens
- [ ] Wrong password → 401 `INVALID_CREDENTIALS` (not 404)
- [ ] Non-existent email → 401 `INVALID_CREDENTIALS` (same code — anti-enumeration)
- [ ] Disabled user → 403 `ACCOUNT_DISABLED`
- [ ] Refresh token hash stored in `refresh_tokens` table (verifiable via DB)
- [ ] JWT `exp` claim is 15 minutes from issue time (configurable via env)
