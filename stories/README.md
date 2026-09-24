# Stories — Dogfood Hackathon Platform

> All stories are written for: **Go 1.23** (backend) · **Next.js 14** (frontend) · **PostgreSQL 16** · **Redis 7** · **MinIO**
> Architecture: **Modular Monolith + Hexagonal (Ports & Adapters)**
> See `.agents/rules/engineering-standards.md` for mandatory coding rules before touching any story.

---

## Team

| Handle | Role | Vertical Slice (owns backend + frontend for same domain) |
|--------|------|----------------------------------------------------------|
| **Sujay** | Full-stack | **Auth** + **Submissions** (backend API + frontend pages) |
| **Keerthika** | Full-stack | **Events** + **Judging** (backend API + frontend pages) |
| **Ridhima** | Full-stack | **Teams** + **Voting** + **Admin** (backend API + frontend pages) |

> **Why vertical slices?** Each person builds the API and the page that calls it. You can integrate and test end-to-end yourself without waiting for someone else. No integration bottleneck.

**35 stories total** — Sujay: 11, Keerthika: 12, Ridhima: 12

---

## Story Board Status

| ID | Title | Epic | Owner | Status | Blocks |
|----|-------|------|-------|--------|--------|
| **SCAFFOLD-001** | Initial project scaffold (baseline for all teammates) | Foundation | Sujay | `[ ]` | **ALL** |
| **F-001** | Go project scaffold + Chi router | Foundation | Sujay | `[ ]` | all |
| **F-002** | Database: PostgreSQL 16 + sqlx + migrations | Foundation | Keerthika | `[ ]` | all |
| **F-003** | Redis 7 + MinIO object storage setup | Foundation | Ridhima | `[ ]` | V-001, S-003 |
| **F-004** | JWT middleware + shared error handler | Foundation | Sujay | `[ ]` | A-001 |
| **F-005** | Next.js 14 scaffold + API client + types | Foundation | Ridhima | `[ ]` | FE-A-001 |
| **A-001** | Register + Login + JWT issue | Auth | Sujay | `[ ]` | A-002 |
| **A-002** | Token Refresh + Logout + Profile | Auth | Sujay | `[ ]` | ADM-001 |
| **A-003** | Password Reset flow | Auth | Sujay | `[ ]` | — |
| **E-001** | Create Event + List + Detail | Events | Keerthika | `[ ]` | E-002, E-003, E-004 |
| **E-002** | Update Event + State Transition | Events | Keerthika | `[ ]` | E-003, E-004, T-001 |
| **E-003** | Create Rubric + Add Criteria | Events/Judging | Keerthika | `[ ]` | E-004, J-002 |
| **E-004** | Invite Judges + List Judges | Events | Keerthika | `[ ]` | J-001 |
| **T-001** | Register for Event + Unregister | Teams | Ridhima | `[ ]` | T-002 |
| **T-002** | Create Team + Generate Invite Code | Teams | Ridhima | `[ ]` | T-003 |
| **T-003** | Join Team via Invite Code + Leave Team | Teams | Ridhima | `[ ]` | S-001 |
| **S-001** | Create Submission Draft | Submissions | Sujay | `[ ]` | S-002 |
| **S-002** | Update Submission + Final Submit | Submissions | Sujay | `[ ]` | S-003, J-001 |
| **S-003** | File Upload — MinIO Cover + Attachments | Submissions | Sujay | `[ ]` | S-004 |
| **S-004** | Submission Gallery + Disqualify | Submissions | Sujay | `[ ]` | V-001, V-002 |
| **J-001** | Judge Queue + Assignment Detail | Judging | Keerthika | `[ ]` | J-002 |
| **J-002** | Submit Scores for Assignment | Judging | Keerthika | `[ ]` | J-003, J-004 |
| **J-003** | Recuse from Assignment | Judging | Keerthika | `[ ]` | J-004 |
| **J-004** | Score Normalization + Ranking | Judging | Keerthika | `[ ]` | V-001, V-002, ADM-003 |
| **V-001** | Cast Vote + Retract Vote | Voting | Ridhima | `[ ]` | V-002 |
| **V-002** | Voting Results Leaderboard | Voting | Ridhima | `[ ]` | ADM-003 |
| **ADM-001** | User Management + Impersonation | Admin | Ridhima | `[ ]` | ADM-002 |
| **ADM-002** | Audit Log — Immutable Activity Trail | Admin | Ridhima | `[ ]` | ADM-003 |
| **ADM-003** | Certificate Generation + Verification | Admin | Ridhima | `[ ]` | — |
| **FE-A-001** | Auth Pages — Login, Register, Logout | Auth (FE) | Sujay | `[ ]` | FE-E-001 |
| **FE-E-001** | Event Listing + Event Detail Pages | Events (FE) | Keerthika | `[ ]` | FE-T-001 |
| **FE-E-002** | Organizer Dashboard — Create + Manage | Events (FE) | Keerthika | `[ ]` | FE-J-001 |
| **FE-T-001** | Team Registration + Management Flow | Teams (FE) | Ridhima | `[ ]` | FE-S-001 |
| **FE-S-001** | Submission Editor + Gallery | Submissions (FE) | Sujay | `[ ]` | FE-J-001, FE-V-001 |
| **FE-J-001** | Judge Interface — Queue + Scoring | Judging (FE) | Keerthika | `[ ]` | FE-V-001 |
| **FE-V-001** | Voting UI + Results Leaderboard | Voting (FE) | Ridhima | `[ ]` | — |

---

## Story Count per Person

| Person | Backend | Frontend | Total |
|--------|---------|----------|-------|
| **Sujay** | SCAFFOLD-001, F-001, F-004, A-001, A-002, A-003, S-001, S-002, S-003, S-004 | FE-A-001, FE-S-001 | **12** |
| **Keerthika** | F-002, E-001, E-002, E-003, E-004, J-001, J-002, J-003, J-004 | FE-E-001, FE-E-002, FE-J-001 | **12** |
| **Ridhima** | F-003, F-005, T-001, T-002, T-003, V-001, V-002, ADM-001, ADM-002, ADM-003 | FE-T-001, FE-V-001 | **12** |

---

## Recommended Build Order (3-Person Parallel)

Dependencies must be respected. Within each phase, all three can work simultaneously.

```
Phase 0 — Foundation (SCAFFOLD-001 first, then F-* in parallel)
  Sujay: SCAFFOLD-001 (initial scaffold — MUST MERGE TO MAIN FIRST)
            then F-001 (Go Chi router setup)  ->  F-004 (JWT middleware)
  Keerthika: [waits for SCAFFOLD-001 to merge, then] F-002 (PostgreSQL + sqlx + migrations)
  Ridhima: [waits for SCAFFOLD-001 to merge, then] F-003 (Redis + MinIO)  +  F-005 (Next.js scaffold)
  ──────────────────────────────────────────────────────────────────────
  ✅ Gate: go build ./cmd/api passes  +  npm run build passes

Phase 1 — Auth (Sujay: full vertical slice)
  Sujay: A-001 → A-002 → A-003 (full auth backend)
            FE-A-001 (auth pages — integrates directly against A-001/A-002)
  Keerthika: E-001 (events create/list — starts in parallel after F-002)
  Ridhima: T-001 (team registration — starts in parallel after F-002, F-003)
  ──────────────────────────────────────────────────────────────────────
  ✅ Gate: Login works end-to-end in browser (page → API → JWT → protected route)

Phase 2 — Events + Teams Core
  Sujay: S-001 → S-002 (submissions draft + submit, needs T-003)
  Keerthika: E-002 → E-003 → E-004 (full events backend)
            FE-E-001 (event listing page — integrates against E-001/E-002)
  Ridhima: T-002 → T-003 (create team + join/leave)
            FE-T-001 (team flow pages — integrates against T-001/T-002/T-003)
  ──────────────────────────────────────────────────────────────────────
  ✅ Gate: Can register, browse events, create team, get invite code — all in browser

Phase 3 — Submissions + Judging + Organizer UI
  Sujay: S-003 → S-004 (file upload + gallery)
            FE-S-001 (submission editor + gallery — integrates against S-001/S-002/S-003)
  Keerthika: J-001 → J-002 → J-003 → J-004 (full judging backend)
            FE-E-002 (organizer dashboard — integrates against E-002/E-003/E-004)
  Ridhima: V-001 → V-002 (voting backend, needs J-004 + S-004)
            ADM-001 → ADM-002 → ADM-003 (admin backend)
  ──────────────────────────────────────────────────────────────────────
  ✅ Gate: Can upload submission cover, organizer can advance event status, rubric created

Phase 4 — Judge UI + Voting UI + Final Integration
  Sujay: [backend done — integration testing + support]
  Keerthika: FE-J-001 (judge scoring UI — integrates against J-001/J-002/J-003)
  Ridhima: FE-V-001 (voting + leaderboard — integrates against V-001/V-002)
  ──────────────────────────────────────────────────────────────────────
  ✅ Final Gate: Full end-to-end in browser:
     register → create team → submit project → judge scores → vote → see leaderboard
```

---

## Definition of Done (per story)

A story is **DONE** when ALL of the following are true:

1. `go test ./internal/{module}/...` → 100% pass (no skips)
2. `go vet ./...` → 0 warnings
3. `go build ./cmd/api` → success
4. `go-arch-lint` → 0 violations
5. All invariants specified in the story are tested with named test functions (`TestXxx_CannotYyyWhenZzz`)
6. Story's "Definition of Done" checklist is fully checked
7. PR merged to `main`

Frontend stories additionally require:
- `npm run build` → 0 TypeScript errors
- `npm run lint` → 0 errors

---

## Invariant Reference (see docs/invariants.md for full list)

| ID | Summary | Where Enforced |
|----|---------|----------------|
| I1 | One team per user per event | DB UNIQUE + use case |
| I2 | One submission per team per event | DB UNIQUE + use case |
| I3 | Register only during `registration_open` + before deadline | Use case |
| I4 | Team size ≤ event maxTeamSize | Use case |
| I5 | One vote per user per submission | DB UNIQUE + use case |
| I6 | Create team only during `registration_open` | Use case |
| I7 | audit_log is append-only (no UPDATE/DELETE) | DB REVOKE |
| I8 | Join/leave team only during `registration_open` | Use case |
| I9 | Submission only during `submissions_open` | Use case |
| I10 | Only admin/organizer can assign judges | Use case |
| I11 | No edit/upload after submission deadline | Domain method |
| I12 | Scoring/voting only within window | Use case |
| I13 | Only `draft` submissions can be finalized | Domain method |
| I14 | Rubric weights must sum to 1.0 | Domain method + DB CHECK |
| I15 | Event status follows state machine | Domain method |
| I16 | Judge can only score their own assignment | Use case + DB lookup |
| I17 | ALL write actions produce audit log | Use case (required) |
| I18 | IP addresses stored as SHA-256 hash only | Domain function |
| I19 | Cannot unregister if already on a team | Use case |

---

## File Locations

```
stories/
├── README.md                          ← this file
├── epic-0-foundation/
│   ├── SCAFFOLD-001-initial-scaffold.md   ← START HERE: Sujay does this first
│   ├── F-001-scaffold.md
│   ├── F-002-database.md
│   ├── F-003-redis-minio.md
│   ├── F-004-jwt-middleware.md
│   └── F-005-nextjs-scaffold.md
├── epic-1-auth/
│   ├── A-001-register-login.md
│   ├── A-002-refresh-logout.md
│   ├── A-003-password-reset.md
│   └── FE-A-001-auth-pages.md
├── epic-2-events/
│   ├── E-001-create-list-detail.md
│   ├── E-002-update-status.md
│   ├── E-003-rubric.md
│   ├── E-004-judges.md
│   ├── FE-E-001-event-pages.md
│   └── FE-E-002-organizer-ui.md
├── epic-3-teams/
│   ├── T-001-register-event.md
│   ├── T-002-create-team.md
│   ├── T-003-join-leave.md
│   └── FE-T-001-team-flow.md
├── epic-4-submissions/
│   ├── S-001-create-draft.md
│   ├── S-002-update-submit.md
│   ├── S-003-file-upload.md
│   ├── S-004-gallery-disqualify.md
│   └── FE-S-001-submission-ui.md
├── epic-5-judging/
│   ├── J-001-queue-detail.md
│   ├── J-002-submit-scores.md
│   ├── J-003-recuse.md
│   ├── J-004-normalize.md
│   └── FE-J-001-judge-ui.md
├── epic-6-voting/
│   ├── V-001-cast-vote.md
│   ├── V-002-results.md
│   └── FE-V-001-voting-ui.md
└── epic-7-admin/
    ├── ADM-001-users-impersonate.md
    ├── ADM-002-audit-log.md
    └── ADM-003-certificates.md
```
