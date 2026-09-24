---
trigger: always_on
---

# Engineering Standards — Dogfood Hackathon Platform
# Auto-loaded by AI agents on every session for this workspace.

## MANDATORY: Read Before Writing Any Code

This project follows **Modular Monolith + Hexagonal Architecture (Ports and Adapters)**.
Tech stack: **Go 1.23 · Chi v5 · pgx/sqlx · PostgreSQL 16 · Redis 7 · MinIO · Next.js 14**
Before brainstorming, scaffolding, or writing any feature, you MUST follow these rules.
Violation of these rules is a build failure (`go-arch-lint` enforces them in CI).

---

## 1. The Non-Negotiable Source Files

Read these before implementing any feature:
- `MASTER-CONTEXT.md` — project rules, 19 invariants (I1–I19), locked tech stack
- `docs/data-model.md` — 19 PostgreSQL tables, full schema
- `docs/invariants.md` — exactly which layer enforces each invariant
- `docs/architecture.md` — hexagonal layers, adapter map, request lifecycle
- `docs/api-design.md` — every endpoint's request/response contract
- `docs/tech-stack.md` — every library and why it was chosen

---

## 2. Module Structure (Modular Monolith)

The codebase is split into **7 bounded-context modules**. Each module is independent.

```
internal/
├── auth/           User, RefreshToken — Register, Login, JWT
├── events/         Event, Track, EventRole — CRUD, state machine
├── teams/          Team, TeamMember — Create, Join, Leave
├── submissions/    Submission, Upload — Draft, Submit, Files
├── judging/        JudgeAssignment, Score, Rubric — Scoring, Normalization
├── voting/         Vote — Cast, Results
└── admin/          AuditLog, Certificate — Admin ops
```

### Within each module, the layers are:
```
internal/{module}/
├── domain/         ← Entities, Value Objects, Enums, Domain Services (PURE GO — zero imports from other layers)
├── port/
│   ├── in.go       ← Use Case interfaces (inbound ports)
│   └── out.go      ← Repository/Service interfaces (outbound ports)
├── usecase/        ← Use Case implementations (orchestration, starts transactions)
├── handler/        ← HTTP handlers using Chi (reads HTTP, calls use case, writes response)
└── repository/     ← pgx/sqlx implementations of out ports (ALL SQL lives here)
```

Additional adapter implementations live in `internal/shared/`:
```
internal/shared/
├── cache/          ← Redis implementations (RateLimiter, Cache)
└── storage/        ← MinIO implementation (FileStorage)
```

---

## 3. The Dependency Rule (NEVER violate this)

Dependencies flow **inward only**. Enforced by `go-arch-lint`.

```
handler/ → usecase/ → domain/
           ↑          ↑
           cannot import outward
```

**Allowed:**
- `handler` imports from `port/in.go` (calls use cases via interfaces)
- `handler` imports from `domain` (uses domain enums/value objects in DTOs)
- `usecase` imports from `domain` (uses entities, calls domain functions)
- `usecase` imports from `port/out.go` (calls repositories via interfaces)
- `repository` imports from `domain` (converts domain structs to/from SQL rows)

**NEVER allowed (build will fail):**
- `domain` imports from `usecase`, `handler`, or `repository`
- `usecase` imports from `handler` or `repository` (only via `port/out.go` interface)
- A `handler` directly calls a `repository` (must go through use case)
- `domain` structs have any framework-specific annotations or tags (no `db:` tags)

**Cross-module rules:**
- Module A may only depend on Module B through an **interface defined in Module A's `port/out.go`**
- Module A may NEVER import Module B's `handler`, `repository`, or `domain` directly
- Cross-module wiring is done in `cmd/api/main.go` only

**Correct cross-module example:**
```go
// teams/port/out.go — teams module defines what it needs from events module
type EventReader interface {
    FindSummaryBySlug(ctx context.Context, slug string) (*EventSummary, error)
}

// cmd/api/main.go — wire events repo as teams' EventReader
teamsSvc := usecase.NewRegisterService(eventsRepo, teamsRepo)
```

---

## 4. What Belongs in Each Layer

### Domain Layer — ZERO external imports allowed

```go
// CORRECT — pure Go, no imports from other layers or framework packages
package domain

import (
    "errors"
    "fmt"
    "time"
    "github.com/google/uuid"
)

type Event struct {
    ID     uuid.UUID
    Status EventStatus
}

func (e *Event) Transition(target EventStatus) error {
    if !validTransition(e.Status, target) {
        return fmt.Errorf("%w: %s → %s", ErrInvalidTransition, e.Status, target)
    }
    e.Status = target
    return nil
}

// WRONG — framework import in domain
import "github.com/jmoiron/sqlx"   // NEVER in domain/
import "github.com/go-chi/chi/v5"   // NEVER in domain/
```

The domain layer contains:
- Structs for entities and value objects (pure Go, no `db:` struct tags)
- Enums and constants (type aliases on string/int)
- Domain functions and methods that encode business rules
- Sentinel errors (`var ErrInvalidTransition = errors.New(...)`)

### Use Case Layer — Orchestration only, uses ports

```go
// CORRECT — depends on port interfaces, never concrete implementations
package usecase

type CastVoteService struct {
    votes       port.VoteRepository    // interface from port/out.go
    rateLimiter port.RateLimiter       // interface from port/out.go
    auditLog    port.AuditLogWriter    // interface from port/out.go
    events      port.EventVotingReader // interface from port/out.go
}

func (s *CastVoteService) Cast(ctx context.Context, cmd port.CastVoteCommand) (*port.VoteDTO, error) {
    // 1. Rate limit check via port (Redis under the hood — use case doesn't know)
    // 2. I12: voting window check via domain function
    // 3. Persist via port (PostgreSQL under the hood — use case doesn't know)
    // 4. Write audit log via port
}

// WRONG — concrete repository in use case
type CastVoteService struct {
    pgRepo *repository.PgVoteRepository // NEVER import the concrete adapter
}
```

Use cases:
- Accept commands/queries as plain Go structs (no HTTP concepts)
- Call domain functions for business rule validation
- Coordinate persistence, cache, and storage calls via ports
- Begin database transactions when needed (passed via `context.Context` or `*sqlx.Tx`)

### Handler Layer — ALL HTTP concepts live here

```go
// CORRECT — HTTP in handler, delegates to use case interface
package handler

type VoteHandler struct {
    cast port.CastVoteUseCase // interface from port/in.go
}

func (h *VoteHandler) CastVote(w http.ResponseWriter, r *http.Request) {
    slug := chi.URLParam(r, "slug")
    callerID := middleware.GetUserID(r.Context())

    var body castVoteRequest
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        response.BadRequest(w, r, "VALIDATION_ERROR", "invalid request body")
        return
    }

    result, err := h.cast.Cast(r.Context(), port.CastVoteCommand{
        VoterID:      callerID,
        EventSlug:    slug,
        SubmissionID: body.SubmissionID,
        RawIP:        r.RemoteAddr, // handler extracts IP — use case hashes it
    })
    if err != nil {
        response.HandleDomainError(w, r, err)
        return
    }
    response.Created(w, r, result)
}
```

### Repository Layer — ALL SQL lives here

```go
// CORRECT — sqlx for simple CRUD, raw SQL with named args for complex queries
package repository

func (r *VoteRepository) Save(ctx context.Context, vote *domain.Vote) error {
    const q = `INSERT INTO votes (id, submission_id, voter_id, event_id, ip_hash, created_at)
               VALUES (:id, :submission_id, :voter_id, :event_id, :ip_hash, :created_at)`
    _, err := r.db.NamedExecContext(ctx, q, toVoteRow(vote))
    return err
}

// For complex queries with JOINs — use raw SQL directly (no ORM)
func (r *SubmissionRepository) ListForEvent(ctx context.Context, eventID uuid.UUID, page, size int) ([]*SubmissionRow, int, error) {
    const q = `
        SELECT s.*, t.name AS team_name, tr.name AS track_name, COUNT(*) OVER() AS total
        FROM submissions s
        LEFT JOIN teams t ON t.id = s.team_id
        LEFT JOIN tracks tr ON tr.id = s.track_id
        WHERE s.event_id = $1 AND s.status = 'submitted'
        ORDER BY s.created_at DESC
        LIMIT $2 OFFSET $3`
    // ...
}
```

---

## 5. Invariant Enforcement — Where Each One Lives

NEVER enforce an invariant in the wrong layer. See `docs/invariants.md` for the full list.

| Invariant | Enforced In | Go Mechanism |
|-----------|-------------|--------------|
| I1, I2, I5 | Database | `UNIQUE` constraint — catch `*pgconn.PgError` code `23505` → return 409 |
| I3, I4, I6, I8, I9, I12, I16, I19 | Use Case layer | Guard check before repository call |
| I7 | Database | `REVOKE UPDATE, DELETE ON audit_log FROM app_user` (migration 022) |
| I10 | Use Case layer | Role lookup before mutation |
| I11, I13 | Domain method | `Submission.CheckCanEdit()`, `Submission.Submit()` |
| I14 | Domain function + DB | `Rubric.ValidateWeights()` + `CHECK (weight > 0)` constraint |
| I15 | Domain method | `Event.Transition(target)` — explicit allowed-transition table |
| I17 | Use Case layer | Every mutating use case calls `auditLog.Write()` (port) |
| I18 | Domain function | `domain.HashIP(rawIP, salt)` — called in use case before saving |

---

## 6. Error Handling — Standard Error Codes

ALL errors return:
```json
{ "error": { "code": "MACHINE_READABLE_CODE", "message": "Human readable" }, "meta": { "requestId": "uuid", "timestamp": "ISO8601" } }
```

Define sentinel errors in `internal/shared/response/errors.go`:

```go
var (
    ErrNotFound          = errors.New("not found")
    ErrForbidden         = errors.New("forbidden")
    ErrUnauthorized      = errors.New("unauthorized")
    ErrInvariantViolated = errors.New("invariant violation")
    ErrDeadlinePassed    = errors.New("deadline passed")
    ErrInvalidTransition = errors.New("invalid state transition")
    ErrRateLimited       = errors.New("rate limited")
)
```

Map errors in `handler/response/handler.go` using `errors.Is()`:

| Sentinel / pgError | HTTP Status | Error Code |
|-------------------|-------------|------------|
| `ErrInvalidTransition` | 422 | `INVALID_STATE_TRANSITION` |
| `ErrDeadlinePassed` | 422 | `DEADLINE_PASSED` |
| `ErrInvariantViolated` | 422 | `INVARIANT_VIOLATION` |
| `pgconn.PgError` code `23505` | 409 | `DUPLICATE_RESOURCE` |
| `ErrForbidden` | 403 | `FORBIDDEN` |
| `ErrUnauthorized` | 401 | `UNAUTHORIZED` |
| `ErrRateLimited` | 429 | `RATE_LIMITED` |
| `ErrNotFound` | 404 | `NOT_FOUND` |

**NEVER** catch errors silently. **NEVER** return HTTP 200 for an error condition.

---

## 7. API Response Shape — Always Use the Envelope

```json
// Success single object
{ "data": { ... }, "meta": { "requestId": "uuid", "timestamp": "ISO8601" } }

// Success list
{ "data": [...], "meta": { "requestId": "...", "page": 1, "totalPages": 5, "totalCount": 48 } }

// Error
{ "error": { "code": "...", "message": "..." }, "meta": { "requestId": "...", "timestamp": "..." } }
```

NO endpoint returns a naked object or naked array. Always wrap in `data`.
Use `response.OK(w, r, data)`, `response.Created(w, r, data)`, `response.Error(w, r, ...)` helpers.

---

## 8. Testing Rules

```
Domain tests:    Plain Go test (testing package). No containers. No mocks. Must run in <100ms.
Use Case tests:  Plain Go test with mock ports (testify/mock or hand-written stubs). No containers.
Repository tests: testify/suite + Testcontainers-go (real PostgreSQL 16). NOT sqlite/in-memory.
Handler tests:   httptest.NewRecorder + full Chi router. Mocked use case interfaces.
```

Every new invariant MUST have a test proving it is enforced.
Test name format: `TestXxxService_CannotYyyWhenZzz` — e.g., `TestSubmissionService_CannotEditAfterDeadline`

Run before every commit:
```sh
go test ./...          # all tests
go vet ./...           # static analysis
go build ./cmd/api     # compilation check
go-arch-lint           # architecture rule check
```

---

## 9. SQL Rules

```
Use sqlx NamedExecContext / GetContext / SelectContext for:
    Simple CRUD (single-table INSERT, UPDATE, DELETE, SELECT by PK or index)

Use raw SQL strings with $1/$2 placeholders or named args for:
    Any JOIN, window function (RANK() OVER, COUNT() OVER), CTE, complex WHERE, direct-to-DTO projection

NEVER use an ORM (GORM, Ent, etc.) — raw SQL only
NEVER load full domain structs for read-only list endpoints — project directly to DTO row structs
NEVER execute SQL in domain/ or usecase/ — only in repository/
NEVER use database/sql directly — always use pgx (via sqlx) for named args and struct scanning
```

---

## 10. What NOT to Build

```
GORM or any ORM          — raw SQL only (sqlx + pgx)
Kafka / RabbitMQ         — Redis pub/sub + goroutines sufficient for this system
Microservices            — modular monolith is the ceiling; do not split deployables
GraphQL                  — REST only
Multiple databases       — one PostgreSQL, one Redis, one MinIO
External OAuth           — custom JWT only (see MASTER-CONTEXT.md)
SQLite / in-memory DB    — always use real PostgreSQL via Testcontainers-go in tests
gRPC                     — REST only; acceptance suite tests REST endpoints
init() functions         — use explicit constructor functions (NewXxx)
Global mutable state     — everything injected via constructor
```

---

## 13. Code Style — Non-Negotiable Rules

```
NO emojis anywhere       — not in code, comments, log messages, error strings, commit
                           messages, or variable names. Zero tolerance. Use plain words.

NO magic numbers         — all numeric constants must be named constants or config values
NO commented-out code    — delete dead code; git history preserves it
NO fmt.Println           — use structured logging (slog) everywhere
NO panic()               — return errors; panics are only acceptable in main() for
                           startup failures (missing required env vars)
NO TODO/FIXME in merged  — all TODOs must be resolved before merging to main
```

---

## 11. go-arch-lint Reference

`go-arch-lint.yml` at the project root enforces all rules above automatically.
Run `go-arch-lint` before any commit. If it fails, the feature is not done.

Key rules enforced:
- `internal/{module}/domain` may NOT import any package outside `internal/{module}/domain` or stdlib
- `internal/{module}/usecase` may NOT import `internal/{module}/handler` or `internal/{module}/repository`
- `internal/{module}/handler` may NOT import `internal/{module}/repository`
- No module may import another module's `domain`, `handler`, or `repository` directly

---

## 12. Quick Layer Decision Guide

- "Does this code read from `r *http.Request` or write to `w http.ResponseWriter`?" → `handler/` only
- "Does this code call `db.QueryContext` or `sqlx.GetContext`?" → `repository/` only
- "Does this code call a Redis or MinIO client?" → `internal/shared/cache/` or `internal/shared/storage/` only
- "Does this code coordinate multiple steps of a business operation?" → `usecase/`
- "Does this code express a business rule or validate a domain concept?" → `domain/`
- "Does this code start a transaction?" → `usecase/` (pass `*sqlx.Tx` via context or explicit param)
- "Does this code import `github.com/go-chi/chi`?" → `handler/` only

**The PostgreSQL swap test:** Ask "what would change if we switched from PostgreSQL to MongoDB?"
That change should touch ONLY `repository/`.
If your answer includes `domain/` or `usecase/`, you've violated the dependency rule.
