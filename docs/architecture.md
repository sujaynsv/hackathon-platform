# SYSTEM ARCHITECTURE
# Dogfood Hackathon Platform

> **Updated**: September 20, 2026 — backend is Go 1.23 + Chi v5
> **Architecture Style**: Hexagonal Architecture (Ports and Adapters) per module
> **Backend**: Go 1.23 + Chi v5 + pgx v5 + sqlx (see tech-stack.md)
> **Source of Truth**: MASTER-CONTEXT.md (19 invariants, 14 entities, state machines)
> **Enforcement**: Go's import system (circular imports = compiler error) + go-arch-lint

---

## How to Read This Document

This architecture document is intentionally **language-agnostic**. Every pattern described here
applies whether you implement in Java, Go, Rust, or Python. The _why_ of every decision is
documented so that if you change language, you change nothing about the design.

The document is organized in four layers:

1. **Big Picture** — what the system looks like from the outside (containers, topology)
2. **Internal Structure** — how code is organized inside the API (layers, packages)
3. **Cross-Cutting Concerns** — auth, rate limiting, audit, error handling
4. **Domain Logic** — how each critical flow (state machines, normalization, voting) works

---

## Part 1: Big Picture — Container Topology

### 1.1 Service Map

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Docker Compose                              │
│                        (dogfood_net bridge)                         │
│                                                                     │
│  ┌──────────┐     ┌───────────────────────────────────────────┐    │
│  │  Client  │────▶│  nginx:alpine  (port 80)                  │    │
│  │ Browser  │     │  Reverse proxy, static files, SSL term.   │    │
│  └──────────>     └──────────┬────────────────┬───────────────┘    │
│                               │                │                    │
│                    /api/*     │     /*         │                    │
│                               ▼                ▼                    │
│              ┌────────────────────┐  ┌─────────────────────┐       │
│              │  api (port 8080)   │  │  web (port 3000)    │       │
│              │  Go 1.23 + Chi v5  │  │  Next.js 14 (SSR)   │       │
│              │  Business logic    │  │  React, TanStack Q  │       │
│              │  All invariants    │  │  Tailwind CSS       │       │
│              └─────┬──────┬───────┘  └─────────────────────┘       │
│                    │      │                                          │
│         ┌──────────┘      └──────────────────────┐                 │
│         ▼                                        ▼                 │
│  ┌────────────────┐                    ┌──────────────────┐        │
│  │  postgres:16   │                    │  redis:7-alpine  │        │
│  │  Primary store │                    │  Rate limits     │        │
│  │  19 tables     │                    │  JWT blacklist   │        │
│  │  All structure │                    │  Job pub/sub     │        │
│  └────────────────┘                    └──────────────────┘        │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────┐      │
│  │  minio/minio  (port 9000 internal, 9001 console)         │      │
│  │  avatars | covers | banners | certificates               │      │
│  └──────────────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────────────┘
```

### 1.2 Startup Order & Health Checks

All startup dependencies are encoded in `docker-compose.yml` `depends_on` with `condition:
service_healthy`. Nothing starts before its dependency is proven ready.

| Order | Service | Health Check Command | Healthy When |
|-------|---------|----------------------|--------------|
| 1 | `postgres` | `pg_isready -U dogfood` | Exit 0 |
| 2 | `redis` | `redis-cli ping` | Returns `PONG` |
| 3 | `minio` | `curl -f http://localhost:9000/minio/health/live` | HTTP 200 |
| 4 | `api` | `curl -f http://localhost:8080/actuator/health` | `{"status":"UP"}` |
| 4 | `web` | `curl -f http://localhost:3000/` | HTTP 200 |
| 5 | `nginx` | depends_on: api, web healthy | — |

**On first boot (`api` container):**
1. golang-migrate runs all pending migrations (001–022) — idempotent, skips applied
2. Seed runner checks if fixture data exists — inserts if not (idempotent)
3. Go binary opens HTTP port 8080 (Chi server)
4. `/health` returns `{"status":"up"}`

---

## Part 2: Internal Structure — Hexagonal Architecture (Ports and Adapters)

### 2.1 Why Hexagonal Architecture?

Alistair Cockburn's Hexagonal Architecture (1994, popularized in 2005) is the dominant modern
pattern for testable, maintainable API services. Its core rule:

> **The application core must not depend on any framework, database, or infrastructure.**

This means: your business logic (event state machines, normalization math, invariant enforcement)
is pure Go structs with no Chi router, no SQL, no Redis. It can be tested with
plain Go unit tests — no containers, no mocks for infrastructure.

```
┌───────────────────────────────────────────────────────────────┐
│                     ADAPTERS (Driving)                        │
│   REST Controllers | CLI | Test Drivers | Admin Console       │
└──────────────────────────────┬────────────────────────────────┘
                                │  calls  ↓  via Inbound Ports (interfaces)
┌───────────────────────────────▼────────────────────────────────┐
│                     APPLICATION CORE                           │
│                                                                │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │  DOMAIN LAYER (innermost)                               │  │
│  │  - Entities: Event, Team, Submission, Score, Vote ...   │  │
│  │  - Value Objects: EventStatus, SubmissionStatus ...     │  │
│  │  - State Machines: EventStateMachine, SubStateMachine   │  │
│  │  - Domain Services: NormalizationService, VoteService   │  │
│  │  - Invariants I1–I19 enforced here                      │  │
│  │  - No framework dependencies. Pure logic.               │  │
│  └─────────────────────────────────────────────────────────┘  │
│                                                                │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │  APPLICATION LAYER                                      │  │
│  │  - Use Cases (Commands + Queries)                       │  │
│  │  - Outbound Port Interfaces:                            │  │
│  │      EventRepository, SubmissionRepository              │  │
│  │      FileStorage, CacheStore, JobQueue                  │  │
│  │  - Transaction boundaries                               │  │
│  └─────────────────────────────────────────────────────────┘  │
└──────────────────────────────┬─────────────────────────────────┘
                                │  calls  ↓  via Outbound Ports (interfaces)
┌───────────────────────────────▼────────────────────────────────┐
│                     ADAPTERS (Driven)                          │
│   PostgreSQL Repos | Redis Cache | MinIO Storage | Job Runner  │
└───────────────────────────────────────────────────────────────┘
```

**The Dependency Rule** (inward-only, never outward):
```
REST Controller → Application Use Case → Domain Service → (nothing)
                                       ↓ calls
                            Repository Interface (Port)
                                       ↓ implemented by
                            PostgreSQL Adapter (Adapter)
```

The domain layer never sees database drivers, Redis clients, or MinIO clients.
It sees only the interfaces it defines. This makes unit-testing invariants trivial.

---

### 2.2 Package Structure — Go Modules

The code is organized into **7 modules** inside `internal/`. Each module owns its domain,
its use cases, and its adapters. Go's import system enforces module boundaries —
a circular import is a compile error. Cross-module calls happen only via `port/in.go` interfaces.

```
dogfood/
│
├── cmd/api/
│   └── main.go                     ← Wire all modules together, start Chi server
│
├── internal/
│   ├── auth/                       ← Module: Authentication & JWT
│   │   ├── domain/
│   │   │   └── user.go             User, RefreshToken structs + domain methods
│   │   ├── port/
│   │   │   ├── in.go               RegisterUseCase, LoginUseCase, RefreshUseCase interfaces
│   │   │   └── out.go              UserRepository, PasswordHasher, TokenIssuer, PasswordValidator, EmailSender, CaptchaValidator interfaces
│   │   ├── usecase/
│   │   │   ├── register.go         RegisterService implements port.RegisterUseCase
│   │   │   ├── login.go            LoginService
│   │   │   └── refresh.go          RefreshService, LogoutService
│   │   ├── handler/
│   │   │   └── handler.go          Chi HTTP handlers — inbound adapter
│   │   └── repository/
│   │       └── postgres.go         pgx/sqlx — outbound adapter
│   │
│   ├── events/                     ← Module: Event Management
│   │   ├── domain/
│   │   │   ├── event.go            Event struct, EventStatus enum, state machine methods
│   │   │   └── track.go            Track struct
│   │   ├── port/
│   │   │   ├── in.go               CreateEventUseCase, TransitionEventUseCase interfaces
│   │   │   └── out.go              EventRepository, TrackRepository interfaces
│   │   ├── usecase/
│   │   │   ├── create.go           CreateEventService
│   │   │   └── transition.go       TransitionEventService (I15)
│   │   ├── handler/
│   │   │   └── handler.go
│   │   └── repository/
│   │       └── postgres.go         sqlx for CRUD, raw SQL for JOIN queries
│   │
│   ├── teams/                      ← Module: Team Management
│   │   ├── domain/
│   │   │   ├── team.go             Team struct + invite code generation
│   │   │   └── member.go           TeamMember struct, role enum
│   │   ├── port/
│   │   │   ├── in.go               CreateTeamUseCase, JoinTeamUseCase, LeaveTeamUseCase
│   │   │   └── out.go              TeamRepository
│   │   ├── usecase/
│   │   ├── handler/
│   │   └── repository/
│   │
│   ├── submissions/                ← Module: Project Submissions + File Uploads
│   │   ├── domain/
│   │   │   ├── submission.go       Submission struct, SubmissionStatus enum, state machine
│   │   │   └── upload.go           Upload struct
│   │   ├── port/
│   │   │   ├── in.go               SubmitProjectUseCase, UpdateDraftUseCase, DisqualifyUseCase
│   │   │   └── out.go              SubmissionRepository, UploadRepository, FileStorage
│   │   ├── usecase/
│   │   ├── handler/
│   │   └── repository/
│   │
│   ├── judging/                    ← Module: Judging + Normalization + Rubrics
│   │   ├── domain/
│   │   │   ├── assignment.go       JudgeAssignment struct, AssignmentStatus
│   │   │   ├── score.go            Score struct, weighted score calculation
│   │   │   └── rubric.go           Rubric, RubricCriterion, weight validation (I14)
│   │   ├── port/
│   │   │   ├── in.go               SubmitScoreUseCase, TriggerNormalizationUseCase, CreateRubricUseCase
│   │   │   └── out.go              ScoreRepository, AssignmentRepository, RubricRepository, EventSignaler
│   │   ├── usecase/
│   │   ├── handler/
│   │   └── repository/
│   │
│   ├── voting/                     ← Module: Community Voting
│   │   ├── domain/
│   │   │   └── vote.go             Vote struct, eligibility checks (I5, I12)
│   │   ├── port/
│   │   │   ├── in.go               CastVoteUseCase
│   │   │   └── out.go              VoteRepository, CacheStore
│   │   ├── usecase/
│   │   ├── handler/
│   │   └── repository/
│   │
│   ├── admin/                      ← Module: Admin + Audit Log + Certificates
│   │   ├── domain/
│   │   │   ├── audit.go            AuditLogEntry struct (append-only, I7)
│   │   │   └── certificate.go      Certificate struct, HMAC-SHA256 hash
│   │   ├── port/
│   │   │   ├── in.go               ImpersonateUseCase, IssueCertificateUseCase, GetAuditLogUseCase
│   │   │   └── out.go              AuditLogRepository, CertificateRepository
│   │   ├── usecase/
│   │   ├── handler/
│   │   └── repository/
│   │
│   └── shared/                     ← Shared infrastructure (no business logic)
│       ├── middleware/
│       │   ├── auth.go             JWT validation middleware
│       │   ├── role.go             Event-scoped role guard
│       │   ├── ratelimit.go        Sliding window Redis rate limiter
│       │   ├── audit.go            Auto-write audit log for POST/PATCH/DELETE
│       │   └── logging.go          Structured slog logging per request
│       ├── response/
│       │   └── response.go         ApiResponse[T], ApiError, Created/OK/NotFound helpers
│       ├── config/
│       │   └── config.go           envconfig struct
│       ├── db/
│       │   └── postgres.go         pgx pool init, migrations runner
│       ├── cache/
│       │   └── redis.go            go-redis client, JWT blacklist, rate limit Lua script
│       └── port/
│           └── storage.go          FileStorage interface (shared across modules)
│
├── migrations/
│   ├── 001_create_users.up.sql
│   ├── 001_create_users.down.sql
│   └── ... 022_revoke_privileges.{up,down}.sql
│
├── go.mod
├── go.sum
├── Makefile                        ← make build, make test, make migrate, make docker
└── docker-compose.yml
```

**Architecture enforcement in Go:**
Go's compiler prevents circular imports — if `usecase/` imports `repository/`, it's a compile error.
`go-arch-lint` enforces the remaining rules: handler/ must not import repository/, domain/ must not
import chi or pgx. Add to CI: `go run github.com/fe3dback/go-arch-lint@latest check`.

---

## Part 3: Cross-Cutting Concerns

### 3.1 The Request Lifecycle

Every API call traverses this chain in order:

```
HTTP Request
     │
     ▼
┌─────────────────────────────────────────────────────┐
│ 1. NGINX (reverse proxy)                            │
│    - Routes /api/* → api:8080                       │
│    - Routes /* → web:3000                           │
│    - Strips /api prefix before forwarding           │
└──────────────────────────┬──────────────────────────┘
                            │
┌──────────────────────────▼──────────────────────────┐
│ 2. REQUEST ID FILTER                                │
│    - Generates UUID v4 per request                  │
│    - Injects into MDC (mapped diagnostic context)   │
│    - Adds X-Request-ID response header              │
└──────────────────────────┬──────────────────────────┘
                            │
┌──────────────────────────▼──────────────────────────┐
│ 3. JWT AUTH FILTER                                  │
│    - Extracts Bearer token from Authorization header │
│    - Verifies HS256 signature (HMAC with JWT_SECRET) │
│    - Checks Redis: EXISTS revoked:{jti}             │
│      → If yes: 401 Unauthorized (token blacklisted) │
│    - Checks: user.is_active = true                  │
│    - Injects UserPrincipal into SecurityContext     │
│    - Injects user_id into MDC for structured logs   │
└──────────────────────────┬──────────────────────────┘
                            │
┌──────────────────────────▼──────────────────────────┐
│ 4. ROLE GUARD (Method Security / Custom Filter)     │
│    - Route groups require specific roles:           │
│       /api/v1/judge/*   → requires JUDGE role       │
│       /api/v1/admin/*   → requires ADMIN flag       │
│       /api/v1/events/*/organize → requires ORGANIZER│
│    - Roles are event-scoped: checks event_roles     │
│      table for (user_id, event_id, role)            │
│    - 403 Forbidden if role not held                 │
└──────────────────────────┬──────────────────────────┘
                            │
┌──────────────────────────▼──────────────────────────┐
│ 5. RATE LIMIT FILTER                                │
│    - Per-user + per-IP sliding window (Redis ZSET)  │
│    - Vote endpoint: 1 req/s per user                │
│    - Score endpoint: 10 req/s per judge             │
│    - Auth endpoints: 5 req/15min per IP             │
│    - 429 Too Many Requests if exceeded              │
└──────────────────────────┬──────────────────────────┘
                            │
┌──────────────────────────▼──────────────────────────┐
│ 6. CONTROLLER (Inbound Adapter)                     │
│    - Deserializes JSON body → request DTO           │
│    - Validates DTO (Bean Validation)                │
│    - Calls Use Case (Application layer)             │
└──────────────────────────┬──────────────────────────┘
                            │
┌──────────────────────────▼──────────────────────────┐
│ 7. USE CASE / APPLICATION SERVICE                   │
│    - Begins transaction boundary (if write op)      │
│    - Loads aggregate from Repository Port           │
│    - Calls domain service / state machine           │
│    - Persists changes via Repository Port           │
│    - Writes AuditLog entry (unconditional for writes)│
│    - Returns response DTO                           │
└──────────────────────────┬──────────────────────────┘
                            │
┌──────────────────────────▼──────────────────────────┐
│ 8. RESPONSE SERIALIZER                              │
│    - Standard JSON envelope:                        │
│       { "data": {...}, "meta": { "requestId": "..." } } │
│    - Error envelope:                                │
│       { "error": { "code": "DEADLINE_PASSED",       │
│                    "message": "..." } }             │
└──────────────────────────┬──────────────────────────┘
                            │
                            ▼
                       HTTP Response
```

### 3.2 Standardized Response Shape

All API responses share one envelope. No exceptions.

**Success:**
```json
{
  "data": { ... },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-09-14T07:40:11Z"
  }
}
```

**Error:**
```json
{
  "error": {
    "code": "INVARIANT_VIOLATION",
    "message": "Submission cannot be edited after deadline",
    "details": { "deadline": "2026-09-12T23:59:00Z" }
  },
  "meta": { "requestId": "...", "timestamp": "..." }
}
```

**Error code taxonomy (machine-readable, never just HTTP status):**

| Code | HTTP Status | Invariant |
|------|-------------|-----------|
| `UNAUTHORIZED` | 401 | JWT invalid/missing |
| `TOKEN_REVOKED` | 401 | JWT in blacklist |
| `FORBIDDEN` | 403 | Role not held (I10, I17) |
| `DEADLINE_PASSED` | 422 | I4 (submission), I6 (score) |
| `DUPLICATE_VOTE` | 409 | I5 (DB unique constraint) |
| `DUPLICATE_TEAM` | 409 | I1 |
| `INVALID_STATE_TRANSITION` | 422 | I15 |
| `NORMALIZATION_REQUIRED` | 422 | I16 |
| `WEIGHTS_DO_NOT_SUM` | 422 | I14 |
| `RATE_LIMITED` | 429 | Rate limiter |
| `INTERNAL_ERROR` | 500 | Unexpected |

---

## Part 4: Domain Logic — The Hard Parts

### 4.1 State Machine Pattern (Invariant I15)

The event status machine is implemented as an explicit **transition table** in the domain layer.
The transition table is the single source of truth — no `if/else` chains, no booleans on entity.

```
Valid transitions:
  draft              → registration_open
  registration_open  → submissions_open
  submissions_open   → judging
  judging            → voting          (optional — organizer may skip)
  judging            → results_published  (if voting skipped)
  voting             → results_published
  results_published  → archived

PROHIBITED: any backwards transition, any skip (e.g., draft → judging)
```

**Implementation pattern (language-agnostic pseudocode):**

```
ALLOWED_TRANSITIONS = {
  draft:              [registration_open],
  registration_open:  [submissions_open],
  submissions_open:   [judging],
  judging:            [voting, results_published],
  voting:             [results_published],
  results_published:  [archived],
  archived:           []
}

function transitionEvent(event, targetStatus, actor):
  allowed = ALLOWED_TRANSITIONS[event.status]
  if targetStatus not in allowed:
    throw StateMachineException("Cannot transition from X to Y")

  // Additional guards for specific transitions:
  if targetStatus == judging:
    validate_I13(event)  // all submissions have ≥1 judge assigned

  if targetStatus == results_published:
    validate_I16(event)  // normalization_status == 'normalized'

  event.status = targetStatus
  auditLog.write(actor, "event.status_changed", event.id, {from: old, to: targetStatus})
  return event
```

The same pattern applies to the **Submission** state machine (draft → submitted → disqualified).

### 4.2 Normalization Algorithm (Z-Score)

The normalization job is the most mathematically sensitive piece. It must be:
- **Deterministic**: same scores always produce same normalized output
- **Idempotent**: safe to re-run (overwrites `normalized_score`)
- **Edge-case safe**: handles σ=0 (all scores identical)
- **Transactional**: either fully applied or fully rolled back

```
NORMALIZATION ALGORITHM (per judging event):

1. PRE-CONDITION CHECK (I8):
   - event.judging_deadline_at < NOW()
     OR COUNT(completed assignments) == COUNT(total assignments)
   - If neither: reject with NORMALIZATION_NOT_READY

2. STATE TRANSITION:
   - event.normalization_status = 'normalizing'
   (prevents duplicate normalization runs)

3. PER-JUDGE STATISTICS:
   For each judge j who has scored at least one criterion:
     μ_j  = MEAN(all raw_scores WHERE judge_id = j)
     σ_j  = POPULATION_STDDEV(all raw_scores WHERE judge_id = j)

     Edge case: σ_j == 0 (judge gave identical scores to everything):
       → Set normalized_score = 0 for all their scores
       → Flag judge in normalization report

4. NORMALIZE:
   For each score s scored by judge j:
     s.normalized_score = (s.raw_score - μ_j) / σ_j
     (or 0 if σ_j = 0)

5. FINAL SCORE PER SUBMISSION:
   For each non-disqualified submission S:
     For each completed JudgeAssignment A on S:
       judge_weighted_score(A) = SUM(
         s.normalized_score × criterion.weight
         FOR EACH score s in assignment A
       )
     final_score(S) = AVG(judge_weighted_score(A) FOR EACH completed A)

6. RANKINGS:
   Within each track: ORDER BY final_score DESC → assign rank
   Overall: ORDER BY final_score DESC → assign overall_rank

7. COMPLETION:
   - event.normalization_status = 'normalized'
   - PUBLISH to Redis channel 'events:normalization' message: {event_id}
   - Organizer can now preview before publishing

8. ROLLBACK:
   - If any step fails: set event.normalization_status = 'failed'
   - Full database transaction wrapping steps 3–7
```

**Why not use the ORM for this?**
The Z-score computation touches every score row for an event in a single pass.
This MUST be a single SQL CTE with window functions — executing it row-by-row in
application code would be O(N) round trips to the database. The raw SQL CTE executed via
`db.ExecContext()` handles this as a single atomic SQL statement.

### 4.3 Voting Architecture (The Voting Rush Problem)

The voting window is the highest-concurrency scenario: potentially hundreds of users
clicking vote simultaneously. The system must:
1. Never let the same user vote twice (I5)
2. Hide vote counts until window closes (I12)
3. Rate-limit voting to prevent ballot stuffing
4. Hash IP addresses before storing (privacy)

```
VOTE FLOW:

1. Client POSTs to /api/v1/votes with { submission_id }
2. Rate limit check: 1 vote/10s per user_id (Redis sliding window)
3. Fetch event → verify event.status == 'voting'
4. Hash client IP: SHA-256(ip + salt) — stored in vote.ip_hash, never raw IP
5. INSERT INTO votes (submission_id, voter_id, event_id, ip_hash)
   → DB unique constraint (voter_id, submission_id) is the ultimate guard (I5)
   → ON CONFLICT: return 409 DUPLICATE_VOTE (not a server error)
6. Return 201 Created { "voted": true }
   NOTE: NEVER return current vote count in this response (I12)

VOTE COUNT VISIBILITY:
  - During voting window: vote_count = NULL (not returned) in all API responses
  - After voting window closes (event.voting_closes_at < NOW()):
    vote_count is returned in gallery and results endpoints
```

**Why is DB constraint (I5) the ultimate guard, not application logic?**
Under concurrent load, two requests for the same (voter, submission) pair can arrive
simultaneously at two different threads/pods. Application-level duplicate checks using
read-then-write are susceptible to TOCTOU (time-of-check-time-of-use) race conditions.
The database `UNIQUE` constraint enforces serializability — only one INSERT wins.
The application treats the resulting unique constraint violation as a 409, not a 500.

### 4.4 Judge Score Isolation (Invariant I9)

Judges must never see each other's scores during the judging window. This is enforced
in the query layer, not just the presentation layer:

```
SCORE QUERY RULES:

During judging window (event.judging_deadline_at > NOW()):
  SELECT * FROM scores
  WHERE assignment_id = ?
    AND judge_id = :current_user_id    ← MANDATORY filter, always applied
  // Never returns other judges' rows, even if caller is admin

After judging window / results_published:
  SELECT * FROM scores
  WHERE submission_id = ?
  // Returns all judges' scores for aggregation/display
```

The `judge_id = :current_user_id` filter is applied **inside the Repository implementation**,
not in the controller. The controller never has the opportunity to omit it.

---

## Part 5: Invariant Enforcement Map

A complete map of where each invariant lives in the architecture:

| ID | Rule | Layer | Mechanism |
|----|------|-------|-----------|
| **I1** | One team per user per event | **DB** | `UNIQUE(user_id, event_id)` on `team_members` |
| **I2** | One submission per team | **DB** | `UNIQUE(team_id, event_id)` on `submissions` |
| **I3** | Judge ≠ own team | **Use Case** | `JudgeAssignmentService` queries `team_members` before INSERT |
| **I4** | No submission edits after deadline | **Use Case** | `SubmitProjectService.update()` checks `event.submission_deadline_at` |
| **I5** | One vote per user per submission | **DB** | `UNIQUE(voter_id, submission_id)` on `votes` |
| **I6** | No score updates after judging deadline | **Use Case** | `SubmitScoreService.submit()` checks `event.judging_deadline_at` |
| **I7** | AuditLog append-only | **DB** | `REVOKE UPDATE, DELETE ON audit_log FROM app_user` (migration V022) |
| **I8** | Normalize only after judging complete | **Use Case** | `NormalizationService.trigger()` pre-condition check |
| **I9** | Judges can't see each other's scores | **Repository** | `ScoreRepository.findByAssignment()` always adds `judge_id = :currentUser` filter |
| **I10** | Organizer cannot score | **Filter Chain** | Role guard rejects organizer-only principals on `/judge/*` routes |
| **I11** | Disqualified excluded from gallery | **Repository** | All public submission queries add `WHERE status != 'disqualified'` |
| **I12** | Vote counts hidden during window | **Use Case** | `GalleryService.getSubmission()` nulls out `vote_count` if window open |
| **I13** | ≥1 judge before judging opens | **State Machine** | Pre-condition in `transitionEvent(→ judging)` |
| **I14** | Rubric weights sum to 1.0 | **Domain + DB** | `RubricValidationService` validates; DB CHECK constraint |
| **I15** | Event state machine | **Domain** | `EventStateMachine.transition()` — explicit transition table |
| **I16** | Publish requires normalization | **State Machine** | Pre-condition in `transitionEvent(→ results_published)` |
| **I17** | Participants blocked from judge routes | **Filter Chain** | Role guard — `/judge/*` requires `JUDGE` role |
| **I18** | Admin impersonation logged | **Filter Chain** | Impersonation filter writes `audit_log` unconditionally before proxying |
| **I19** | Can't unregister with submitted project | **Use Case** | `UnregisterService.execute()` checks submission status |

---

## Part 6: Data Flow Diagrams

### 6.1 Registration → Submission (Flow F1)

```
Participant                  API                     PostgreSQL             MinIO
    │                         │                           │                   │
    ├──POST /auth/register────▶│                           │                   │
    │                         ├──INSERT users─────────────▶│                   │
    │◀─201 { access_token }───│                           │                   │
    │                         │                           │                   │
    ├──POST /events/{slug}/register──▶│                   │                   │
    │                         ├──INSERT event_registrations▶│                  │
    │◀─201 { registration }───│                           │                   │
    │                         │                           │                   │
    ├──POST /teams────────────▶│                           │                   │
    │                         ├──INSERT teams, team_members▶│                  │
    │◀─201 { team, invite_code }──│                       │                   │
    │                         │                           │                   │
    ├──POST /submissions───────▶│                          │                   │
    │                         ├──INSERT submissions(status=draft)──▶│          │
    │◀─201 { submission }─────│                           │                   │
    │                         │                           │                   │
    ├──PUT /submissions/{id}/cover (multipart)──▶│        │                   │
    │                         ├──validate MIME (magic bytes)          │       │
    │                         ├──PutObject─────────────────────────────────────▶│
    │                         ├──INSERT uploads────────────▶│                  │
    │◀─200 { coverUrl }───────│                           │                   │
    │                         │                           │                   │
    ├──POST /submissions/{id}/submit──▶│                  │                   │
    │                         ├──CHECK deadline (I4)──────▶│                  │
    │                         ├──UPDATE status=submitted──▶│                  │
    │◀─200 { submitted: true }│                           │                   │
```

### 6.2 Normalization Trigger (Flow F5)

```
Organizer                  API                  PostgreSQL          Redis
    │                       │                       │                 │
    ├──POST /events/{slug}/normalize──▶│            │                 │
    │                       ├──CHECK I8 (deadline)──▶│               │
    │                       ├──UPDATE norm_status='normalizing'──▶│   │
    │◀─202 Accepted─────────│                       │                 │
    │                       │                       │                 │
    │              [async normalization job]         │                 │
    │                       ├──BEGIN TRANSACTION─────▶│               │
    │                       ├──CTE: compute μ,σ──────▶│               │
    │                       ├──UPDATE scores.normalized──▶│            │
    │                       ├──UPDATE submissions.final_score──▶│      │
    │                       ├──UPDATE submissions.rank──▶│             │
    │                       ├──COMMIT──────────────────▶│              │
    │                       ├──UPDATE norm_status='normalized'──▶│     │
    │                       ├──PUBLISH events:normalization {id}───────▶│
    │                       │                       │                 │
    ├──GET /events/{slug}/results/preview──▶│       │                 │
    │◀─200 { preview scores }───────────────│       │                 │
    │                       │                       │                 │
    ├──POST /events/{slug}/results/publish──▶│      │                 │
    │                       ├──CHECK I16 (norm complete)──▶│          │
    │                       ├──UPDATE event.status='results_published'──▶│
    │◀─200 { published: true }──────────────│       │                 │
```

---

## Part 7: Security Architecture

### 7.1 Defense in Depth Model

The system uses layered security. Each layer is independent — bypassing one does not
compromise another.

```
Layer 1: Transport (Nginx)
  - Strip sensitive headers (Server, X-Powered-By)
  - Rate limit at edge level (basic, before application)

Layer 2: Authentication (JWT Filter)
  - Signature verification (HMAC-HS256, shared secret)
  - Expiry check (exp claim)
  - Blacklist check (Redis O(1) lookup)
  - User active check (is_active = true in DB)

Layer 3: Authorization (Role Guard)
  - Event-scoped role check per route group
  - is_admin flag for platform-wide operations

Layer 4: Application (Use Case / Domain)
  - Business-rule invariants (I1–I19)
  - Deadline enforcement
  - State machine guards
  - HIBP compromised password validation (for Auth)

Layer 5: Database
  - UNIQUE constraints (I1, I2, I5)
  - REVOKE on audit_log (I7)
  - CHECK constraints (I14 rubric weights)
  - Row-level security (future: could add RLS policies)
```

### 7.2 IP Address Handling (Privacy by Design)

Raw IP addresses are NEVER stored. The vote flow illustrates the pattern:

```
received_ip = request.getRemoteAddr()  // e.g., "203.0.113.42"
ip_hash = SHA-256(received_ip + IP_HASH_SALT)
// Only ip_hash is stored in votes.ip_hash and audit_log.ip_address
```

`IP_HASH_SALT` is a secret env var. Without it, stored hashes cannot be reversed to IPs.
This protects participant privacy while still allowing anti-stuffing analysis.

### 7.3 AuditLog Immutability (I7)

```sql
-- Migration V022 (runs after all table creation)
-- The app_user role used by the API has INSERT only, no UPDATE or DELETE
REVOKE UPDATE, DELETE ON TABLE audit_log FROM app_user;

-- Even if application code has a bug and tries to UPDATE audit_log,
-- the database will reject it with a permission error.
```

The `AuditLog` write happens in the **Use Case layer**, not in a middleware afterthought.
Every state-changing use case explicitly calls `auditLogRepository.append(entry)` as the
last step before returning. This ensures it's inside the transaction — logged or rolled back.

---

## Part 8: API Route Map

Complete route inventory, organized by middleware requirements:

```
Public (no auth required):
  GET  /api/v1/events                        List published events
  GET  /api/v1/events/{slug}                 Get event detail
  GET  /api/v1/events/{slug}/gallery         Public gallery (status filter, I11)
  GET  /api/v1/certificates/verify/{hash}    Certificate verification
  POST /api/v1/auth/register                 Create account
  POST /api/v1/auth/login                    Issue JWT pair
  POST /api/v1/auth/refresh                  Refresh access token

Authenticated (any valid JWT):
  POST /api/v1/auth/logout                   Revoke current token
  GET  /api/v1/me                            Current user profile
  PUT  /api/v1/me                            Update profile

Participant (authenticated + registered for event):
  POST /api/v1/events/{slug}/register        Register for event
  DELETE /api/v1/events/{slug}/register      Unregister (I19 guard)
  POST /api/v1/events/{slug}/teams           Create team
  POST /api/v1/events/{slug}/teams/join      Join team by invite_code
  GET  /api/v1/teams/{id}                    Get team detail
  POST /api/v1/submissions                   Create draft submission
  GET  /api/v1/submissions/{id}              Get own submission
  PUT  /api/v1/submissions/{id}              Update (I4 guard)
  PUT  /api/v1/submissions/{id}/cover        Upload cover image
  POST /api/v1/submissions/{id}/submit       Submit (locks submission)
  POST /api/v1/votes                         Cast vote (I5, I12, rate-limit)

Judge (authenticated + JUDGE role for event):
  GET  /api/v1/judge/queue                   Judge's assignment queue
  GET  /api/v1/judge/assignments/{id}        Get assignment + submission
  POST /api/v1/judge/assignments/{id}/scores Submit scores (I3, I6, I9)
  POST /api/v1/judge/assignments/{id}/recuse Recuse from assignment

Organizer (authenticated + ORGANIZER role for event):
  POST /api/v1/events                        Create event
  PUT  /api/v1/events/{slug}                 Update event config
  POST /api/v1/events/{slug}/tracks          Add track
  POST /api/v1/events/{slug}/rubrics         Create rubric
  POST /api/v1/events/{slug}/rubrics/{id}/criteria  Add criterion (I14)
  POST /api/v1/events/{slug}/status          Advance state machine (I15)
  POST /api/v1/events/{slug}/judges/invite   Invite judge
  POST /api/v1/events/{slug}/assignments     Assign judges (I3, I13)
  POST /api/v1/events/{slug}/normalize       Trigger normalization (I8)
  GET  /api/v1/events/{slug}/results/preview Preview scores
  POST /api/v1/events/{slug}/results/publish Publish (I16)
  POST /api/v1/submissions/{id}/disqualify   Disqualify (I11)

Admin (is_admin = true on user):
  GET  /api/v1/admin/users                   List all users
  PUT  /api/v1/admin/users/{id}              Update user (deactivate)
  POST /api/v1/admin/impersonate/{userId}    Impersonate (I18 — always logged)
  GET  /api/v1/admin/audit-log              Browse audit trail
```

---

## Part 9: Modern Architecture Patterns Applied

This section explains the system design patterns used and why they were chosen over alternatives.

### 9.1 Vertical Slice vs. Layered — What We Actually Use

We use **Hexagonal Architecture** (Ports & Adapters), which is a form of layered architecture
but organized by **component boundaries** (domain-driven), not technical layers.

The key difference from traditional three-tier layering:

```
Traditional three-tier (WRONG for us):
  controller/ → service/ → repository/
  All events, all submissions, all teams in the same technical bucket.

Hexagonal (what we use):
  domain/ → usecase/ → repository/
  One vertical slice per bounded context. Boundaries enforced by Go package imports and go-arch-lint.
```

This makes it easy to find all code related to "the vote feature" — it's in one vertical
stack, not spread across three horizontal packages.

### 9.2 Repository Pattern — Why Interfaces Between Use Cases and DB

Every database operation goes through a port interface:

```go
// Port — defined in port/out.go, no database knowledge
type SubmissionRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*domain.Submission, error)
    Save(ctx context.Context, sub *domain.Submission) error
    FindByEventID(ctx context.Context, eventID uuid.UUID, statuses ...domain.SubmissionStatus) ([]*domain.Submission, error)
}

// Adapter — implemented in repository/postgres.go using sqlx
type PgSubmissionRepository struct {
    db *sqlx.DB
}
func (r *PgSubmissionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Submission, error) {
    // Uses pgx/sqlx internally — usecase never knows
}
```

**Why this matters:**
- Unit tests for use cases use a trivial in-memory implementation — no Testcontainers needed
- The use case is decoupled from PostgreSQL specifics (JSONB, partial indexes)
- If we add a Redis read-through cache, we wrap the adapter — use case unchanged

### 9.3 CQRS Lite — Separate Read and Write Paths

We apply a pragmatic subset of CQRS (Command Query Responsibility Segregation) without
an event store or separate databases. The pattern:

```
WRITES (Commands):
  LoadSubmission(id)           → domain object → validate → mutate → save
  Uses sqlx, transactional, returns void or minimal DTO

READS (Queries):
  GetGalleryPage(eventId, page) → raw SQL SELECT directly projected to response DTO row struct
  Does NOT load full domain objects — goes straight from DB to DTO
  No transaction needed — read-only, optimized for speed
```

**Why this matters for normalization and gallery:**
- Gallery queries join submissions + vote_counts + rankings — complex aggregation.
  Loading 500 full domain objects and mapping them is O(N) work.
  A single raw SQL query with the exact columns projected to a DTO struct is O(1) queries, full optimization.
- The Z-score normalization is a pure CTE with a bulk UPDATE — never touches domain objects.

### 9.4 Anti-Corruption Layer (ACL) — DTO Mapping at Boundaries

Domain objects (entities) NEVER cross the adapter boundary. Every adapter maps to/from DTOs.

```
Inbound (REST → Domain):
  CreateSubmissionRequest (DTO)
    → SubmitProjectService.create(command)
      → Submission (domain entity)

Outbound (Domain → REST):
  Submission (domain entity)
    → SubmissionResponse (DTO) ← mapping done in controller or mapper class
      → JSON
```

This prevents "anemic domain model" syndrome where domain objects become property bags
polluted with `@JsonProperty`, `@Column`, and `@NotNull` annotations. Domain objects
only contain business logic — no serialization or persistence concerns.

### 9.5 Idempotency Pattern for Background Jobs

The normalization job uses status flags to prevent duplicate execution:

```
normalization_status state machine:
  awaiting_judging
    → ready_to_normalize   (all assignments complete)
    → normalizing          (job started — CAS operation: only one transition wins)
    → normalized           (job done — results preview available)
    → failed               (job failed — organizer can retry)
```

The `normalizing` state acts as a distributed lock. Before starting, the job does:
```sql
UPDATE events
SET normalization_status = 'normalizing'
WHERE id = ? AND normalization_status = 'ready_to_normalize'
RETURNING id
```
If this UPDATE returns 0 rows, another process already started normalization. Job aborts.
This is a **compare-and-swap (CAS)** at the DB level — no external lock manager needed.

---

## Part 10: What Makes This Production-Grade

The MASTER-CONTEXT.md says "production-grade, robust, correct." These are not adjectives —
they are specific architectural requirements:

| Requirement | How We Meet It |
|-------------|----------------|
| **Robust under concurrent voting** | Goroutines + DB unique constraint as final guard for I5 |
| **Correct invariant enforcement** | I1–I19 all enforced in domain/DB, not UI. Test suite verifies each. |
| **Append-only audit trail** | DB-level REVOKE. Application cannot delete audit rows even if buggy. |
| **Offline-first** | All services in `docker-compose.yml`. No external API calls. Seeds on first run. |
| **Recoverable failures** | Background goroutines set status='failed' and expose retry endpoint. No silent failures. |
| **No orphaned files** | Every MinIO upload tracked in `uploads` table. Garbage collection is auditable. |
| **Testable invariants** | Domain layer has zero framework dependencies. Every invariant testable with plain `go test`. |
| **Deterministic normalization** | Z-score math is a single idempotent SQL CTE. Re-running overwrites scores cleanly. |
| **JWT revocation** | Redis blacklist with auto-TTL. Logout is instant and verified on every request. |

---

## Part 11: Why Not Kafka or RabbitMQ? — The Streaming Services Decision

This is a deliberate, researched decision — not an oversight. Here is the full analysis.

### 11.1 What Kafka, RabbitMQ, and Redis Actually Are

These three tools solve fundamentally different problems. Using the wrong one is an
architecture smell, not a feature.

| Tool | Primary Model | Core Guarantee | Best Fit |
|------|--------------|----------------|----------|
| **Apache Kafka** | Distributed commit log (append-only) | Messages stored durably, replayable forever, multiple consumers at different offsets | High-throughput event streaming, event sourcing, audit logs at massive scale, CDC |
| **RabbitMQ** | Message broker (AMQP) | Per-message delivery ACK, dead-letter queues, complex routing via exchanges/bindings | Background task workers, reliable job queues, complex fan-out routing |
| **Redis Pub/Sub** | Fire-and-forget channel | Zero persistence — if no subscriber is listening when a message is published, it is lost | Ephemeral real-time signals, transient notifications, "I don't care if it's lost" |
| **Redis Streams** | Persistent log in Redis | Consumer groups, message persistence (memory-bound), at-least-once delivery | Mid-ground: more reliable than Pub/Sub, simpler than Kafka/RabbitMQ |

### 11.2 Kafka — Why We Don't Need It

**Kafka is the right tool when you have:**
- Millions of events/second (we have at most hundreds of votes during a voting window)
- Multiple independent services consuming the same event stream (we have ONE service)
- Event sourcing / replayability requirements (our audit trail is in PostgreSQL)
- Cross-team, cross-service data pipelines (single monolith)

**What Kafka costs in a docker-compose setup:**

```
Apache Kafka (KRaft mode, no ZooKeeper):
  # Adds: 500MB container image + 2GB JVM heap
  # For: Signaling "normalization job completed" to one goroutine
```

**The normalization signal we need:**
```
Producer:  PUBLISH events:normalization "event-uuid-here"   (1 line of Redis)
Consumer:  SUBSCRIBE events:normalization                    (1 line of Redis)
```

Kafka would add ~600MB, 2 containers, and significant operational complexity to signal
one event from one goroutine/thread to one listener. That is the definition of overkill.

**Key insight:** Kafka's value comes from *multiple independent consumers reading the same
stream at their own pace*. We have one producer and one consumer, in the same JVM process.
An in-process `@EventListener` would work just as well.

### 11.3 RabbitMQ — Why We Don't Need It

**RabbitMQ is the right tool when you have:**
- Background jobs that MUST be processed exactly once, with retries on failure
- Dead-letter queues for failed job recovery
- Complex routing: fanout, topic, headers exchanges
- Distributed workers on separate machines consuming from a shared queue

**What we actually need:**
```
Our "job queue" has exactly ONE job type: NormalizationJob.
Our "worker" is a goroutine launched in the same Go binary.
Our "routing" is trivial: one job goes to one handler.
Our "retry" requirement: if normalization fails, the organizer clicks retry.
```

**RabbitMQ would add:**
- `rabbitmq:3.13-management-alpine` container (~200MB image)
- AMQP connection management, channel pooling
- Message serialization/deserialization (Jackson to AMQP)
- Consumer acknowledgment logic, dead-letter queue configuration
- 15-30s startup, health checks, management UI port (15672)

**For what?** To replace `@Async + CompletableFuture<Void>` which is 10 lines of Java.

**The only scenario where RabbitMQ would make sense for us:**
If the normalization job needed to run on a *separate machine* from the API server,
or if it needed guaranteed delivery even if the API server crashed mid-execution.
We don't have that requirement. If the API crashes during normalization, the organizer
sees `normalization_status = 'failed'` and re-triggers. That is sufficient.

### 11.4 The Actual Message Passing in Our System

We have exactly **two async communication needs**:

| Need | Sender | Receiver | Message | Loss Acceptable? | Solution |
|------|--------|----------|---------|------------------|----------|
| Normalization complete signal | `NormalizationJobRunner` | Frontend polling / organizer dashboard | `{event_id}` | **Yes** — frontend polls `/events/{slug}` anyway | Redis Pub/Sub |
| Normalization job trigger | `NormalizationController` | `NormalizationJobRunner` | `{event_id}` | **No** — but it's in-process | goroutine (`go normalize(ctx, eventID)`) |

The normalization job is triggered *in the same process* via a goroutine — no message broker
needed at all. The completion signal is sent via Redis Pub/Sub because it's a simple
"let anyone who cares know it's done" notification. If a frontend tab wasn't subscribed,
it will notice on the next poll.

### 11.5 The Hexagonal Port: EventSignaler

Because we follow Hexagonal Architecture, the messaging mechanism is *behind a port interface*.
This means we can swap Redis Pub/Sub for Kafka or RabbitMQ without touching domain code:

```go
// Port — defined in port/out.go, technology-agnostic
type EventSignaler interface {
    PublishNormalizationComplete(ctx context.Context, eventID uuid.UUID) error
}

// Current adapter — Redis Pub/Sub
type RedisEventSignaler struct {
    client *redis.Client
}

func (s *RedisEventSignaler) PublishNormalizationComplete(ctx context.Context, eventID uuid.UUID) error {
    return s.client.Publish(ctx, "events:normalization", eventID.String()).Err()
}

// Future adapter — if we ever need Kafka (one file change, zero domain change)
type KafkaEventSignaler struct {
    producer *kafka.Producer
}

func (s *KafkaEventSignaler) PublishNormalizationComplete(ctx context.Context, eventID uuid.UUID) error {
    return s.producer.Produce("normalization-complete", eventID.String())
}
```

**This is the power of Hexagonal Architecture:** The domain doesn't care whether the
signal goes via Redis, Kafka, RabbitMQ, or a WebSocket push. The adapter is the only
thing that changes. The port contract (`EventSignaler`) is stable.

---

## Part 12: Complete Hexagonal Architecture — All Adapters Named Precisely

This is the full, project-specific diagram. Every box is a real class/interface you will write.

```
╔══════════════════════════════════════════════════════════════════════════════════╗
║                        INBOUND ADAPTERS (Driving Side)                          ║
║  These receive external input and translate it into Use Case calls              ║
╠══════════════════════════════════════════════════════════════════════════════════╣
║                                                                                  ║
║  ┌─────────────────────┐  ┌──────────────────────┐  ┌──────────────────────┐   ║
║  │  HTTP Handlers      │  │  Chi Middleware       │  │  Background Goroutine│   ║
║  │  (Chi Router)       │  │  Chain               │  │  (go func())         │   ║
║  │                     │  │                      │  │                      │   ║
║  │ AuthHandler         │  │ RequestIDMiddleware   │  │ NormalizationJob     │   ║
║  │ EventHandler        │  │ JWTMiddleware         │  │ Runner               │   ║
║  │ SubmissionHandler   │  │ RoleMiddleware        │  │                      │   ║
║  │ JudgeHandler        │  │ RateLimitMiddleware   │  │ (triggered by go     │   ║
║  │ VoteHandler         │  │ AuditMiddleware       │  │  func() in usecase)  │   ║
║  │ AdminHandler        │  │                      │  │                      │   ║
║  └──────────┬──────────┘  └──────────┬───────────┘  └──────────┬───────────┘   ║
╚═════════════╪══════════════════════╪════════════════════════════╪════════════════╝
              │                      │                            │
              │  calls via           │  calls via                 │ calls via
              ▼                      ▼                            ▼
╔═════════════════════════════════════════════════════════════════════════════════╗
║                         INBOUND PORTS (Driving Ports)                           ║
║  Interfaces that the application core exposes to the outside world              ║
╠═════════════════════════════════════════════════════════════════════════════════╣
║                                                                                 ║
║  RegisterUseCase          SubmitScoreUseCase      CastVoteUseCase               ║
║  LoginUseCase             AssignJudgeUseCase      PublishResultsUseCase         ║
║  RegisterForEventUseCase  CreateRubricUseCase     TriggerNormalizationUseCase   ║
║  CreateTeamUseCase        DisqualifyUseCase       ImpersonateUserUseCase        ║
║  SubmitProjectUseCase     TransitionEventUseCase  IssueCertificateUseCase       ║
║                                                                                 ║
╠═════════════════════════════════════════════════════════════════════════════════╣
║                                                                                 ║
║  ──────────────────── APPLICATION CORE (The Hexagon) ───────────────────────── ║
║                                                                                 ║
║  ┌───────────────────────────────────────────────────────────────────────────┐  ║
║  │                        DOMAIN LAYER                                       │  ║
║  │  (ZERO framework imports — pure Go structs and logic only)            │  ║
║  │                                                                           │  ║
║  │  AGGREGATES & ENTITIES:                                                   │  ║
║  │    Event          Submission      Team          Score                     │  ║
║  │    TeamMember     JudgeAssignment Vote          Certificate               │  ║
║  │    RubricCriterion AuditLogEntry  User          Track                     │  ║
║  │                                                                           │  ║
║  │  VALUE OBJECTS:                                                           │  ║
║  │    EventStatus    SubmissionStatus  AssignmentStatus                      │  ║
║  │    NormalizationStatus  IpHash  VerificationHash                          │  ║
║  │                                                                           │  ║
║  │  STATE MACHINES:                                                          │  ║
║  │    EventStateMachine       → enforces I15 (valid transition graph)        │  ║
║  │    SubmissionStateMachine  → draft/submitted/disqualified                 │  ║
║  │                                                                           │  ║
║  │  DOMAIN SERVICES:                                                         │  ║
║  │    NormalizationDomainService  → Z-score formula (pure math, no SQL)      │  ║
║  │    RubricValidationService     → I14 (weights must sum to 1.0)            │  ║
║  │    JudgeConflictService        → I3 (judge ≠ own team)                   │  ║
║  │    VoteEligibilityService      → I5, I12 checks                           │  ║
║  │                                                                           │  ║
║  │  DOMAIN EXCEPTIONS:                                                       │  ║
║  │    InvariantViolationException  DeadlinePassedException                   │  ║
║  │    StateMachineException        UnauthorizedRoleException                 │  ║
║  └───────────────────────────────────────────────────────────────────────────┘  ║
║                                                                                 ║
║  ┌───────────────────────────────────────────────────────────────────────────┐  ║
║  │                      APPLICATION LAYER (Use Cases)                        │  ║
║  │                                                                           │  ║
║  │  RegisterService              SubmitScoreService                          │  ║
║  │  LoginService                 AssignJudgeService                          │  ║
║  │  RegisterForEventService      CreateRubricService                         │  ║
║  │  CreateTeamService            DisqualifyService                           │  ║
║  │  SubmitProjectService         TransitionEventService                      │  ║
║  │  CastVoteService              NormalizationApplicationService             │  ║
║  │  PublishResultsService        IssueCertificateService                     │  ║
║  └───────────────────────────────────────────────────────────────────────────┘  ║
║                                                                                 ║
╠═════════════════════════════════════════════════════════════════════════════════╣
║                         OUTBOUND PORTS (Driven Ports)                           ║
║  Interfaces the application core NEEDS — defined inside core, zero impl detail  ║
╠═════════════════════════════════════════════════════════════════════════════════╣
║                                                                                 ║
║  PERSISTENCE PORTS:                                                             ║
║    EventRepository           ← findBySlug, save, findById                      ║
║    SubmissionRepository      ← findById, findByEventId, save, updateStatus      ║
║    TeamRepository            ← findByInviteCode, save, lockTeam                 ║
║    UserRepository            ← findByEmail, save, findById                      ║
║    ScoreRepository           ← findByAssignment, saveAll, findForNormalization   ║
║    VoteRepository            ← save (upsert), countBySubmission                  ║
║    JudgeAssignmentRepository ← findPendingForJudge, save, complete              ║
║    RubricRepository          ← findByEvent, save, validateWeights               ║
║    AuditLogRepository        ← append (INSERT-only, no reads in use cases)      ║
║    CertificateRepository     ← save, findByVerificationHash                     ║
║    RefreshTokenRepository    ← save, findByHash, revokeAllForUser               ║
║                                                                                 ║
║  SERVICE PORTS:                                                                 ║
║    FileStorage               ← upload, presignedUrl, delete                     ║
║    CacheStore                ← setWithTtl, get, delete, exists (blacklist, RL)  ║
║    EventSignaler             ← publishNormalizationComplete                     ║
║    PasswordHasher            ← hash, verify                                     ║
║    TokenIssuer               ← issueAccessToken, issueRefreshToken              ║
║    IpHasher                  ← hash(rawIp, salt) → SHA-256 hex                  ║
║                                                                                 ║
╚═══════════════════════════════════════╤═════════════════════════════════════════╝
                                        │ implemented by
                                        ▼
╔══════════════════════════════════════════════════════════════════════════════════╗
║                      OUTBOUND ADAPTERS (Driven Side)                            ║
║  These implement the ports — the only place infrastructure code lives           ║
╠══════════════════════════════════════════════════════════════════════════════════╣
║                                                                                 ║
║  ┌────────────────────────┐  ┌───────────────────────┐  ┌────────────────────┐ ║
║  │  PERSISTENCE ADAPTERS  │  │  CACHE ADAPTERS       │  │  STORAGE ADAPTER   │ ║
║  │  (PostgreSQL via pgx/  │  │  (Redis via go-redis) │  │  (MinIO SDK)       │ ║
║  │   sqlx — raw SQL only) │  │                       │  │                    │ ║
║  │                        │  │ RedisCache            │  │ MinioFileStorage   │ ║
║  │ PgEventRepository      │  │  implements:          │  │  implements:       │ ║
║  │ PgSubmissionRepository │  │  RateLimiter          │  │  FileStorage       │ ║
║  │ PgUserRepository       │  │  ResultsCache         │  │                    │ ║
║  │ PgScoreRepository      │  │                       │  │  Operations:       │ ║
║  │ PgVoteRepository       │  │  Operations:          │  │  - PutObject       │ ║
║  │ PgAssignmentRepository │  │  - JWT blacklist      │  │  - PresignedUrl    │ ║
║  │ PgRubricRepository     │  │    revoked:{jti}      │  │  - RemoveObject    │ ║
║  │ PgAuditLogRepository   │  │  - Rate limit         │  │                    │ ║
║  │ PgCertificateRepo      │  │    ratelimit:{uid}    │  │  Buckets:          │ ║
║  │ PgRefreshTokenRepo     │  │  - Event role cache   │  │  - uploads/avatars │ ║
║  │                        │  │                       │  │  - uploads/covers  │ ║
║  │ NOTE: Reads (gallery,  │  └───────────────────────┘  │  - uploads/banners │ ║
║  │ rankings) use raw SQL  │                              │  - certificates/   │ ║
║  │ projected to DTO rows  │  ┌───────────────────────┐  └────────────────────┘ ║
║  │ no domain object load  │  │  SIGNALING ADAPTER    │                         ║
║  │                        │  │  (Redis Pub/Sub)      │  ┌────────────────────┐ ║
║  └────────────────────────┘  │                       │  │  SECURITY ADAPTERS │ ║
║                              │ RedisEventSignaler    │  │                    │ ║
║                              │  implements:          │  │ BcryptPasswordHash │ ║
║                              │  EventSignaler        │  │  implements:       │ ║
║                              │                       │  │  PasswordHasher    │ ║
║                              │  Channels:            │  │                    │ ║
║                              │  events:normalization │  │ JWTIssuer          │ ║
║                              │                       │  │  implements:       │ ║
║                              │  Why Redis not Kafka: │  │  TokenIssuer       │ ║
║                              │  - Fire-and-forget OK │  │                    │ ║
║                              │  - One consumer only  │  │ SHA256IpHasher     │ ║
║                              │  - Already in stack   │  │  implements:       │ ║
║                              │  - 1 line vs 50 lines │  │  IpHasher          │ ║
║                              └───────────────────────┘  └────────────────────┘ ║
╚══════════════════════════════════════════════════════════════════════════════════╝

External Systems (connected to by Outbound Adapters):
  ┌──────────────┐  ┌─────────────────┐  ┌──────────────┐
  │ PostgreSQL 16 │  │   Redis 7       │  │  MinIO       │
  │ port 5432     │  │   port 6379     │  │  port 9000   │
  └──────────────┘  └─────────────────┘  └──────────────┘
```

### 12.1 Port-to-Adapter Mapping Table

The complete mapping of every port interface to its concrete adapter class:

| Port Interface | Adapter (Go file) | Technology | Location |
|---------------|--------------|------------|----------|
| `EventRepository` | `EventRepository` (postgres.go) | sqlx + raw SQL | `internal/events/repository/` |
| `SubmissionRepository` | `SubmissionRepository` (postgres.go) | sqlx + raw SQL joins | `internal/submissions/repository/` |
| `TeamRepository` | `TeamRepository` (postgres.go) | sqlx | `internal/teams/repository/` |
| `UserRepository` | `UserRepository` (postgres.go) | sqlx | `internal/auth/repository/` |
| `ScoreRepository` | `ScoreRepository` (postgres.go) | raw SQL CTEs (normalization) | `internal/judging/repository/` |
| `VoteRepository` | `VoteRepository` (postgres.go) | sqlx + ON CONFLICT upsert | `internal/voting/repository/` |
| `JudgeAssignmentRepository` | `AssignmentRepository` (postgres.go) | raw SQL | `internal/judging/repository/` |
| `RubricRepository` | `RubricRepository` (postgres.go) | sqlx | `internal/judging/repository/` |
| `AuditLogRepository` | `AuditLogRepository` (postgres.go) | sqlx INSERT only | `internal/admin/repository/` |
| `CertificateRepository` | `CertificateRepository` (postgres.go) | sqlx | `internal/admin/repository/` |
| `RefreshTokenRepository` | `RefreshTokenRepository` (postgres.go) | sqlx | `internal/auth/repository/` |
| `FileStorage` | `MinioStorage` (minio.go) | minio-go v7 SDK | `internal/shared/storage/` |
| `CacheStore` | `RedisCache` (redis.go) | go-redis/redis v9 | `internal/shared/cache/` |
| `EventSignaler` | `RedisEventSignaler` (redis.go) | Redis Pub/Sub | `internal/shared/cache/` |
| `PasswordHasher` | `BcryptHasher` (bcrypt.go) | golang.org/x/crypto | `internal/shared/auth/` |
| `TokenIssuer` | `JWTIssuer` (jwt.go) | golang-jwt/jwt v5 | `internal/shared/auth/` |
| `IpHasher` | `SHA256IpHasher` (hasher.go) | crypto/sha256 (stdlib) | `internal/shared/auth/` |

### 12.2 sqlx vs Raw SQL Split — The Decision Rule

Not every repository uses the same query style. The rule is:

```
Use sqlx named queries when:
  - Simple CRUD: findByID, save, delete
  - Single-table operations with no complex joins
  - Examples: User, Team, Vote, Certificate, AuditLog

Use raw SQL strings (db.QueryContext) when:
  - Multi-table JOINs (Event + EventRoles + Tracks)
  - Window functions (score normalization, rankings)
  - Conditional logic in SQL (I9: judge_id filter always applied)
  - Pagination with total counts
  - Direct-to-struct projections (gallery, judge queue)
  - Any query where N+1 would occur with multiple round-trips
```

This keeps common operations concise while keeping complex queries explicit and auditable.

### 12.3 What Replaceability Looks Like in Practice

Because every infrastructure dependency is behind a port, you can swap any adapter:

```
TODAY:                          FUTURE (zero domain change):
EventSignaler                   EventSignaler
  └─▶ RedisEventSignaler          └─▶ KafkaEventSignaler
       PUBLISH to Redis                producer.Produce("topic", msg)

FileStorage                     FileStorage
  └─▶ MinioStorage                └─▶ S3Storage
       PutObject to MinIO              s3.PutObject(...)

CacheStore                      CacheStore
  └─▶ RedisCache                  └─▶ LocalCache (for dev)
       Redis ZSET rate limits          sync.Map in-memory

PasswordHasher                  PasswordHasher
  └─▶ BcryptHasher                └─▶ Argon2Hasher
       bcrypt cost 12                   argon2id (OWASP 2024)
```

In every case: **the domain use case code is untouched**. The `main.go` constructor
injection swaps which implementation is injected. This is the **Open-Closed Principle** at
the architecture level.

---

*This document is the authoritative architecture reference.*
*Changes to invariants, state machines, or flows must be reflected here before implementation.*
*See MASTER-CONTEXT.md for the source of all business rules.*
