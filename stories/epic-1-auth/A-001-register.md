---
id: A-001
title: User Registration — POST /api/v1/auth/register
epic: auth
owner: Sujay (backend)
status: "[ ] not-started"
branch: story/A-001-register
blocks: A-002, FE-A-001
blocked-by: F-004
---

# A-001 · User Registration — POST /api/v1/auth/register

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal layer rules, error handling, testing patterns (§3–§8)
- `MASTER-CONTEXT.md` — Invariant **I1**: "A user can only be on one team per event" (relies on unique email for identity)
- `docs/api-design.md §Auth` — exact request/response shape for `POST /api/v1/auth/register`
- `docs/data-model.md §users` — `users` table schema (email UNIQUE constraint enforces one account per email)
- `docs/invariants.md` — no specific invariant for this endpoint, but I1 depends on user identity being unique

## What We're Building

A registration endpoint that creates a new user account. The user provides email, password, and display name. The API hashes the password with bcrypt, stores the user in PostgreSQL, issues a JWT access token (15-min TTL) and a refresh token (7-day TTL), and returns them immediately so the client is logged in right after registration.

**Endpoint:** `POST /api/v1/auth/register`  
**Auth required:** None (public)

**Request:**
```json
{ "email": "alice@example.com", "password": "Password123!", "displayName": "Alice" }
```

**Success response (201 Created):**
```json
{
  "data": {
    "user": { "id": "uuid", "email": "alice@example.com", "displayName": "Alice", "avatarUrl": null, "isAdmin": false, "createdAt": "2026-09-20T10:00:00Z" },
    "accessToken": "eyJ...",
    "refreshToken": "eyJ..."
  },
  "meta": { "requestId": "uuid", "timestamp": "2026-09-20T10:00:00Z" }
}
```

**Error cases:**
| Condition | HTTP | Code |
|-----------|------|------|
| Email already registered | 409 | `DUPLICATE_RESOURCE` |
| Email format invalid | 400 | `VALIDATION_ERROR` |
| Password < 8 chars | 400 | `VALIDATION_ERROR` |
| Display name blank | 400 | `VALIDATION_ERROR` |

## Files to Create

### internal/auth/domain/user.go
```go
package domain

import (
    "errors"
    "strings"
    "time"
    "github.com/google/uuid"
)

// Domain errors — use fmt.Errorf("%w: detail", domain.ErrXxx) to wrap with context.
var (
    ErrEmailTaken   = errors.New("email already registered")
    ErrInvalidEmail = errors.New("invalid email format")
    ErrWeakPassword = errors.New("password too short: minimum 8 characters")
)

type User struct {
    ID           uuid.UUID
    Email        string
    PasswordHash string
    DisplayName  string
    AvatarURL    *string
    IsActive     bool
    IsAdmin      bool
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// NewUser validates and constructs a new User (without saving).
// All validation lives here — not in the handler or use case.
func NewUser(email, displayName string) (*User, error) {
    email = strings.ToLower(strings.TrimSpace(email))
    if !isValidEmail(email) {
        return nil, ErrInvalidEmail
    }
    displayName = strings.TrimSpace(displayName)
    if displayName == "" {
        return nil, errors.New("display name is required")
    }
    now := time.Now().UTC()
    return &User{
        ID:          uuid.New(),
        Email:       email,
        DisplayName: displayName,
        IsActive:    true,
        IsAdmin:     false,
        CreatedAt:   now,
        UpdatedAt:   now,
    }, nil
}

func isValidEmail(email string) bool {
    // Simple check: contains exactly one @ with non-empty parts before and after
    parts := strings.Split(email, "@")
    return len(parts) == 2 && len(parts[0]) > 0 && strings.Contains(parts[1], ".")
}
```

### internal/auth/port/in.go
```go
package port

import "context"

// RegisterUseCase is the inbound port for user registration.
// The handler calls this interface — never the concrete RegisterService struct.
type RegisterUseCase interface {
    Register(ctx context.Context, cmd RegisterCommand) (*AuthResponse, error)
}

type RegisterCommand struct {
    Email       string
    Password    string
    DisplayName string
}

type AuthResponse struct {
    User         UserDTO
    AccessToken  string
    RefreshToken string
}

type UserDTO struct {
    ID          string  `json:"id"`
    Email       string  `json:"email"`
    DisplayName string  `json:"displayName"`
    AvatarURL   *string `json:"avatarUrl"`
    IsAdmin     bool    `json:"isAdmin"`
    CreatedAt   string  `json:"createdAt"`
}
```

### internal/auth/port/out.go
```go
package port

import (
    "context"
    "github.com/dogfood/platform/internal/auth/domain"
    "github.com/google/uuid"
)

// UserRepository is the outbound port for user persistence.
type UserRepository interface {
    Save(ctx context.Context, user *domain.User) error
    FindByEmail(ctx context.Context, email string) (*domain.User, error)
    FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
    ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// PasswordHasher is the outbound port for password hashing.
// Backed by bcrypt adapter — use case never imports golang.org/x/crypto.
type PasswordHasher interface {
    Hash(plain string) (string, error)
    Verify(hash, plain string) bool
}

// TokenIssuer is the outbound port for JWT generation.
type TokenIssuer interface {
    IssueAccessToken(userID, email string, isAdmin bool) (string, error)
    IssueRefreshToken(userID string) (string, error)
}

// RefreshTokenRepository is the outbound port for refresh token persistence.
type RefreshTokenRepository interface {
    Save(ctx context.Context, token *domain.RefreshToken) error
    FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error)
    RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}
```

### internal/auth/usecase/register.go
```go
package usecase

import (
    "context"
    "fmt"

    "github.com/dogfood/platform/internal/auth/domain"
    "github.com/dogfood/platform/internal/auth/port"
    "github.com/dogfood/platform/internal/shared/response"
)

type RegisterService struct {
    users   port.UserRepository
    hasher  port.PasswordHasher
    tokens  port.TokenIssuer
    refresh port.RefreshTokenRepository
}

func NewRegisterService(
    users port.UserRepository,
    hasher port.PasswordHasher,
    tokens port.TokenIssuer,
    refresh port.RefreshTokenRepository,
) *RegisterService {
    return &RegisterService{users: users, hasher: hasher, tokens: tokens, refresh: refresh}
}

func (s *RegisterService) Register(ctx context.Context, cmd port.RegisterCommand) (*port.AuthResponse, error) {
    // 1. Validate password length (domain rule)
    if len(cmd.Password) < 8 {
        return nil, fmt.Errorf("%w", domain.ErrWeakPassword)
    }

    // 2. Check email uniqueness
    exists, err := s.users.ExistsByEmail(ctx, cmd.Email)
    if err != nil {
        return nil, fmt.Errorf("check email: %w", err)
    }
    if exists {
        return nil, fmt.Errorf("%w", response.ErrInvariantViolated) // maps to 409
    }

    // 3. Build domain user (validates email format)
    user, err := domain.NewUser(cmd.Email, cmd.DisplayName)
    if err != nil {
        return nil, err
    }

    // 4. Hash password (via port — bcrypt adapter)
    hash, err := s.hasher.Hash(cmd.Password)
    if err != nil {
        return nil, fmt.Errorf("hash password: %w", err)
    }
    user.PasswordHash = hash

    // 5. Persist user
    if err := s.users.Save(ctx, user); err != nil {
        return nil, fmt.Errorf("save user: %w", err)
    }

    // 6. Issue tokens
    accessToken, err := s.tokens.IssueAccessToken(user.ID.String(), user.Email, user.IsAdmin)
    if err != nil {
        return nil, fmt.Errorf("issue access token: %w", err)
    }
    refreshToken, err := s.tokens.IssueRefreshToken(user.ID.String())
    if err != nil {
        return nil, fmt.Errorf("issue refresh token: %w", err)
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
        RefreshToken: refreshToken,
    }, nil
}
```

### internal/auth/handler/handler.go
```go
package handler

import (
    "encoding/json"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/dogfood/platform/internal/auth/port"
    "github.com/dogfood/platform/internal/shared/response"
)

type AuthHandler struct {
    register port.RegisterUseCase  // interface — NOT *usecase.RegisterService
    // login, refresh, logout added in A-002, A-003
}

func NewAuthHandler(register port.RegisterUseCase) *AuthHandler {
    return &AuthHandler{register: register}
}

func (h *AuthHandler) Routes() chi.Router {
    r := chi.NewRouter()
    r.Post("/register", h.Register)
    return r
}

type registerRequest struct {
    Email       string `json:"email"`
    Password    string `json:"password"`
    DisplayName string `json:"displayName"`
}

func (req registerRequest) validate() error {
    if req.Email == "" { return fmt.Errorf("email is required") }
    if req.Password == "" { return fmt.Errorf("password is required") }
    if req.DisplayName == "" { return fmt.Errorf("displayName is required") }
    return nil
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    var req registerRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.BadRequest(w, r, "VALIDATION_ERROR", "invalid request body")
        return
    }
    if err := req.validate(); err != nil {
        response.BadRequest(w, r, "VALIDATION_ERROR", err.Error())
        return
    }

    result, err := h.register.Register(r.Context(), port.RegisterCommand{
        Email:       req.Email,
        Password:    req.Password,
        DisplayName: req.DisplayName,
    })
    if err != nil {
        response.HandleDomainError(w, r, err)
        return
    }
    response.Created(w, r, result)
}
```

### internal/auth/repository/postgres.go
```go
package repository

import (
    "context"
    "errors"
    "fmt"

    "github.com/jackc/pgx/v5/pgconn"
    "github.com/jmoiern/sqlx"
    "github.com/dogfood/platform/internal/auth/domain"
    "github.com/dogfood/platform/internal/shared/response"
    "github.com/google/uuid"
)

type UserRepository struct {
    db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
    return &UserRepository{db: db}
}

func (r *UserRepository) Save(ctx context.Context, user *domain.User) error {
    const q = `
        INSERT INTO users (id, email, password_hash, display_name, is_active, is_admin, created_at, updated_at)
        VALUES (:id, :email, :password_hash, :display_name, :is_active, :is_admin, :created_at, :updated_at)`
    if _, err := r.db.NamedExecContext(ctx, q, user); err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            return fmt.Errorf("%w: email already registered", response.ErrInvariantViolated)
        }
        return fmt.Errorf("user repo save: %w", err)
    }
    return nil
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
    var count int
    err := r.db.GetContext(ctx, &count, "SELECT COUNT(1) FROM users WHERE email = $1", email)
    return count > 0, err
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
    var user domain.User
    err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE email = $1", email)
    if err != nil {
        return nil, fmt.Errorf("%w: user not found", response.ErrNotFound)
    }
    return &user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
    var user domain.User
    err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", id)
    if err != nil {
        return nil, fmt.Errorf("%w: user not found", response.ErrNotFound)
    }
    return &user, nil
}
```

### Wire in cmd/api/main.go
Add to main.go after database setup:
```go
// Auth module
userRepo    := authrepo.NewUserRepository(database)
hasher      := shared.NewBcryptHasher()
tokenIssuer := shared.NewJWTIssuer(cfg.JWTSecret, cfg.JWTAccessTTLMinutes, cfg.JWTRefreshTTLDays)
refreshRepo := authrepo.NewRefreshTokenRepository(database)
registerSvc := authusecase.NewRegisterService(userRepo, hasher, tokenIssuer, refreshRepo)
authHandler := authhandler.NewAuthHandler(registerSvc)

r.Mount("/api/v1/auth", authHandler.Routes())
```

## Tests Required

### Domain tests (internal/auth/domain/user_test.go)
Plain `go test`, no external dependencies, runs < 50ms:
- `TestNewUser_ValidInput_ReturnsUser` — email normalized to lowercase
- `TestNewUser_EmptyDisplayName_ReturnsError`
- `TestNewUser_InvalidEmail_ReturnsErrInvalidEmail` — e.g. "notanemail"

### Use case tests (internal/auth/usecase/register_test.go)
Mock the port/out.go interfaces with in-memory structs — no DB, no Redis:
- `TestRegisterService_ValidInput_ReturnsAuthResponse`
- `TestRegisterService_DuplicateEmail_ReturnsInvariantViolated` — mock ExistsByEmail returns true
- `TestRegisterService_WeakPassword_ReturnsErrWeakPassword` — password "abc" (< 8 chars)

### Handler tests (internal/auth/handler/handler_test.go)
`httptest.NewRecorder()`, mock `port.RegisterUseCase`:
- `TestRegisterHandler_ValidInput_Returns201` — assert `data.user.email`, `data.accessToken` present
- `TestRegisterHandler_MissingEmail_Returns400` — assert `error.code == "VALIDATION_ERROR"`
- `TestRegisterHandler_DuplicateEmail_Returns409` — mock returns `ErrInvariantViolated`

### Integration test (internal/auth/repository/postgres_test.go)
testcontainers-go, real PostgreSQL 16:
- `TestUserRepository_Save_Success`
- `TestUserRepository_Save_DuplicateEmail_ReturnsErrInvariantViolated`
- `TestRegisterEndpoint_Integration` — full stack: `POST /api/v1/auth/register` → assert 201, parse JWT

## Definition of Done
- [ ] `go test ./internal/auth/...` → 100% green
- [ ] `go vet ./internal/auth/...` → 0 warnings
- [ ] `POST /api/v1/auth/register` with valid body → 201 with `data.user`, `data.accessToken`, `data.refreshToken`
- [ ] Duplicate email → 409 `DUPLICATE_RESOURCE`
- [ ] Missing field → 400 `VALIDATION_ERROR`
- [ ] Password < 8 chars → 400 `VALIDATION_ERROR`
- [ ] All response fields match `docs/api-design.md` exactly (field names, types, casing)
- [ ] No circular imports (`go build ./...` succeeds)
