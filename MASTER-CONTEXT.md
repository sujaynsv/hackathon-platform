# MASTER CONTEXT — Dogfood Hackathon Platform

> **This file is the single source of truth for all planning documents.**
> Every document (PRD, Data Model, Architecture, API Design, Eng Specs) must be consistent with this file.
> When in doubt, this file wins. Update this file when decisions change.

---

## 1. PROJECT RULES (Non-negotiable)

- Starts with `docker compose up` — no cloud, no external APIs, no auth-as-a-service
- Works fully offline with network turned off
- Seeds fixture data on first run automatically
- OSI-approved open source license
- T1 is the minimum to be judged. T2 = judging engine. T3 = voting. T4 = stretch.
- Tier claims verified by acceptance suite (automated tests) — honesty rewarded
- 3-person team building this — philosophy: **production-grade, robust, correct**
- Auth: **custom JWT only** (bcrypt passwords, no OAuth providers, no Clerk, no Auth0)
- A clean correct T2 beats a broken T4.

### Confirmed Tech Stack (FINAL — do not change without team agreement)

> **Updated September 20, 2026**: Backend reverted from Java 21 back to **Go 1.23**.
> All other services, data model, invariants, and API design are unchanged.

| Service | Technology | Role |
|---------|-----------|------|
| Primary DB | **PostgreSQL 16** | All structured/relational data + JSONB semi-structured fields |
| Cache / Ephemeral | **Redis 7** | Rate limiting (sliding window), JWT blacklist, normalization signaling |
| File Storage | **MinIO (latest)** | Cover images, event banners, avatars, generated certificate PDFs — S3-compatible, fully self-hosted |
| Backend API | **Go 1.23 + Chi v5** | HTTP API, business logic, goroutine-based background jobs |
| SQL Layer | **pgx v5 + sqlx** | Native PostgreSQL driver + named struct scanning; no ORM, all SQL explicit |
| Migrations | **golang-migrate v4** | File-based SQL migrations (same SQL files, compatible format) |
| Frontend | **Next.js 14 (TypeScript)** | SSR UI, React, TanStack Query, vanilla CSS + CSS variables (no Tailwind — see `docs/frontend-standards.md`) |
| Proxy | **Nginx (alpine)** | Reverse proxy, static file serving |
| JWT | **golang-jwt/jwt v5** | HS256, 24hr access / 7d refresh |
| Logging | **log/slog (Go stdlib)** | Structured JSON logs to stdout — zero dependency |

**Docker Compose services: `postgres`, `redis`, `minio`, `api`, `web`, `nginx` — all offline, all self-hosted.**

File storage abstracted behind a `FileStorage` interface (`internal/shared/port/storage.go`) so MinIO can be swapped to real AWS S3 with a single env var change in production.


---

## 2. ROLES (Event-scoped except Admin)

| Role | Event-scoped? | Key capability | Key restriction |
|------|-------------|----------------|-----------------|
| `participant` | Yes | Create team, submit project, vote | Cannot see judge scores before publish |
| `judge` | Yes | Score assigned submissions | Cannot see other judges' scores during window |
| `organizer` | Yes | Configure event, assign judges, publish results | Cannot score projects (unless also invited as judge) |
| `admin` | No (platform-global) | Everything across all events | Cannot bypass audit log |

**One user can hold different roles in different events.** Roles are stored as event_role records, not on the user table (except `is_admin` flag on user).

---

## 3. DOMAIN OBJECTS (All 14 entities)

| Entity | Key Fields | Key Constraints |
|--------|-----------|-----------------|
| `User` | id, email, password_hash, display_name, is_verified, is_active, is_admin | email unique |
| `Event` | id, slug, title, status, all date windows, max_team_size, judges_per_submission, voting_weight | slug unique |
| `EventRegistration` | id, event_id, user_id, registered_at, unregistered_at, is_active | (event_id, user_id) unique |
| `Track` | id, event_id, name, prizes (jsonb), is_active | |
| `Team` | id, event_id, name, invite_code, created_by, is_locked | invite_code unique |
| `TeamMember` | id, team_id, user_id, role (owner\|member), joined_at | (user_id, event_id) unique = ONE TEAM PER USER PER EVENT |
| `Submission` | id, team_id, event_id, track_id, title, tagline, description, demo_url, repo_url, video_url, cover_image_url, additional_links(jsonb), status, submitted_at, gallery_order | (team_id, event_id) unique = ONE SUBMISSION PER TEAM |
| `Rubric` | id, event_id, track_id (nullable), name, is_active | |
| `RubricCriterion` | id, rubric_id, name, description, max_score, weight, sort_order | SUM(weight) per rubric MUST = 1.0 |
| `JudgeAssignment` | id, event_id, submission_id, judge_id, assigned_at, assigned_by, status, completed_at | judge cannot be on submission's team |
| `Score` | id, assignment_id, submission_id, judge_id, criterion_id, raw_score, normalized_score (nullable), comment, scored_at | (assignment_id, criterion_id) unique |
| `Vote` | id, submission_id, voter_id, event_id, voted_at, ip_hash | (voter_id, submission_id) unique |
| `Certificate` | id, user_id, event_id, type (participation\|winner\|judge), rank, verification_hash | HMAC-SHA256 |
| `AuditLog` | id, event_id, actor_id, action (enum), target_type, target_id, metadata(jsonb), ip_address, created_at | APPEND-ONLY, no UPDATE/DELETE |

---

## 4. EVENT STATUS MACHINE

```
draft → registration_open → submissions_open → judging → [voting →] results_published → archived
```

- Transitions are time-based (auto) or manually triggered by organizer
- No backwards transitions allowed (I15)
- Cannot skip states (e.g. draft → judging is forbidden)

**Feature access by status:**
- `registration_open` / `submissions_open`: participants can register, join teams
- `submissions_open`: teams can create/edit submissions
- `judging`: judges can score
- `voting`: community voting active, scores hidden
- `results_published`: scores + ranks visible, certificates available

---

## 5. SUBMISSION STATUS MACHINE

```
draft (editable) → submitted (locked) → [disqualified]
```

- `draft → submitted`: triggered by participant, before `submission_deadline_at`
- `submitted → disqualified`: organizer only
- `disqualified → submitted`: organizer reinstatement
- ALL edits blocked after `submission_deadline_at` regardless of status (I4)
- Disqualified submissions excluded from gallery and rankings (I11)

---

## 6. TEAM STATUS

```
forming (invite active) → locked (when submission created OR deadline passes)
```

- Lock is permanent — no new members can join or leave after lock
- `invite_code` becomes invalid when team locks

---

## 7. JUDGE ASSIGNMENT STATUS

```
pending → in_progress → completed
   ↓
 recused (organizer creates new assignment for replacement)
```

---

## 8. NORMALIZATION STATE (event-level)

```
awaiting_judging → ready_to_normalize → normalizing → normalized → previewing → results_published
```

**Z-Score formula (per judge j):**
```
μ_j = MEAN(all raw_scores by judge j)
σ_j = STDDEV(all raw_scores by judge j)
normalized_score_ij = (raw_score_ij - μ_j) / σ_j

Edge case: if σ_j = 0, set normalized_score = 0 and flag judge in report.

final_score_S = AVG(SUM(normalized_score × criterion.weight)) across all completed judges
Rankings within each track: ORDER BY final_score DESC
```

---

## 9. HARD INVARIANTS (All 19 — enforced at DB or service layer, never UI only)

| ID | Rule | Layer |
|----|------|-------|
| I1 | One team per user per event | DB unique: (user_id, event_id) via team_members |
| I2 | One submission per team per event | DB unique: (team_id, event_id) |
| I3 | Judge cannot be on submission's team | Service: check before INSERT assignment |
| I4 | No submission edits after submission_deadline_at | Service: deadline check in update handler |
| I5 | One vote per user per submission | DB unique: (voter_id, submission_id) |
| I6 | No score updates after judging_deadline_at | Service: deadline check in score handler |
| I7 | AuditLog is append-only | DB: app user has no UPDATE/DELETE on audit_log |
| I8 | Normalization only after deadline passed OR all assignments completed | Service: state check |
| I9 | Judges cannot see other judges' scores during judging window | Service: always filter scores by judge_id = current_user |
| I10 | Organizer cannot score projects | Middleware: role guard on /judge/* routes |
| I11 | Disqualified submissions excluded from gallery and rankings | Service: status filter on all public queries |
| I12 | Vote counts hidden until voting window closes | Service: visibility check on vote count queries |
| I13 | Every submitted non-disqualified submission must have ≥1 judge assigned | Service: pre-flight validation before judging window opens |
| I14 | Rubric criterion weights must sum to 1.0 | Service + DB: validated on create/update |
| I15 | Event state machine transitions must follow allowed graph | Service: state machine guard |
| I16 | Results can only be published after normalization complete | Service: state check before publish |
| I17 | Participants cannot access judging endpoints | Middleware: role guard on /judge/* routes |
| I18 | Admin impersonation always logged | Middleware: unconditional audit write |
| I19 | Cannot un-register if user has a submitted (non-draft) submission | Service: check submission status before un-register |

---

## 10. CRITICAL FLOWS (Summary)

| Flow | Key steps |
|------|-----------|
| **F1: Registration → Submission** | Register account → verify email → register for event → create/join team → create draft → edit → submit |
| **F2: Event Setup** | Create event → add tracks → create rubric with criteria (weights = 1.0) → publish |
| **F3: Judge Assignment** | Invite judges → accept invite → manual or auto-assign (round-robin, conflict-aware) |
| **F4: Judging** | View queue → open submission (sees only own past scores) → score each criterion → mark complete |
| **F5: Normalization** | All assignments done → trigger normalize → Z-score per judge → compute final_score → preview → publish |
| **F6: Community Voting** | Voting window opens → randomized gallery (no vote counts) → one vote per user per submission → rate limited → IP hashed |
| **F7: Certificates** | Results published → generate participant/winner/judge certs → HMAC hash → public verify URL |

---

## 11. MIDDLEWARE STACK (Request lifecycle for every API call)

```
1. Auth middleware     — verify JWT, load user (is_active check)
2. Role middleware     — check user's event role for this route group
3. Rate limit         — per-user + per-IP limits
4. Handler            — business logic
5. Audit middleware   — write to AuditLog for all state-changing requests (POST, PATCH, DELETE)
6. Response           — standardized JSON response shape
```

---

## 12. EXPLICIT OUT OF SCOPE

- Real-time WebSocket chat
- Payment processing
- OAuth / SSO login (Google, GitHub)
- Mobile native app
- Multi-language i18n
- Email newsletters
- Video file hosting (URLs only)
- Code execution / sandboxing
- Multi-tenant SaaS (single platform instance)

---

## 13. BONUS CHALLENGES (worth pursuing)

| Bonus | Points | What to implement |
|-------|--------|-------------------|
| Normalization Proof | +5 | Show Z-score effect on fixture data in acceptance report |
| Pairwise Mode | +5 | Bradley-Terry estimator for rankings from pairwise comparisons |
| Threat Model | +3 | Written doc covering Sybil, ballot stuffing, collusion, deadline gaming |
| API First | +3 | OpenAPI spec for every endpoint |

---

## 14. DOCUMENTS IN THIS REPO

| File | Purpose |
|------|---------|
| `docs/README.md` | Index |
| `docs/architecture.md` | System design, service topology, Go package structure (internal/), invariant enforcement |
| `docs/product-requirements.md` | PRD — user stories, acceptance criteria, NFRs, tier checklists |
| `docs/data-model.md` | Full PostgreSQL schema, 19 tables, indexes, normalization SQL, MinIO buckets, migrations |
| `docs/actors.md` | Role capabilities detail |
| `docs/domain-objects.md` | ER diagram + entity field tables |
| `docs/invariants.md` | All 19 invariants with enforcement detail |
| `docs/glossary.md` | 45+ domain terms |
| `docs/state-machines/*.md` | State machine diagrams for each entity |
| `docs/flows/01-07.md` | Sequence diagrams for each user flow |

---

*This file must be read at the start of every planning session before writing any new document.*
*Update this file whenever a decision changes — then update the relevant detailed doc.*
