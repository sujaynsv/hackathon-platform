# Product Requirements Document (PRD)
# Dogfood Hackathon Platform

> **Version**: 1.0 — Pre-spec (will be updated after Sep 24 full spec release)
> **Authors**: Team (2 members)
> **Philosophy**: Production-grade, robust, correct. A clean T2 beats a broken T4.
> **Consistency anchor**: MASTER-CONTEXT.md — all requirements below are derived from it.

---

## 1. Problem Statement

Hackathon Raptors runs 35+ online hackathons per year across 85+ countries. Their current tooling cannot handle the full lifecycle — registration, team formation, submission, judge assignment, score normalization, results, and archiving — in a single self-hostable system.

Existing platforms have converged on the same basic feature set and none are designed for the organizer's real engineering problems: fair judging, role isolation, score normalization, anti-abuse voting, and reliable self-hosted deployment.

**This platform solves that.** The winning implementation will be forked and run in production by Hackathon Raptors for future events.

---

## 2. Goals

| Goal | Success Criterion |
|------|------------------|
| **G1** | Platform starts from `docker compose up` on any laptop with Docker installed |
| **G2** | Platform works fully offline — no network connection required after image pull |
| **G3** | Seeds fixture data automatically on first run |
| **G4** | Supports the complete hackathon lifecycle: registration → judging → results |
| **G5** | Judge scores are fair and normalized across judges with different scoring styles |
| **G6** | Role isolation is enforced at the API layer — not just the UI |
| **G7** | Acceptance suite passes for at least T1 + T2 |
| **G8** | All deliverables present: README, ARCHITECTURE, DATA-MODEL, JUDGING, acceptance-report.txt, demo video |

---

## 3. Non-Goals (Explicit Out of Scope)

| Out of Scope | Rationale |
|-------------|-----------|
| OAuth / SSO login (Google, GitHub) | Custom JWT only — no external auth dependency |
| Real-time WebSocket chat | Async HTTP is sufficient for comments |
| Payment processing | Prizes awarded manually by organizer |
| Mobile native app | Responsive web only |
| Multi-language i18n | English only for hackathon submission |
| Video file hosting | Store URLs only — no video upload |
| Code execution / sandboxing | Not required by spec |
| Email delivery via external SMTP | In-app notifications only; email as optional local SMTP |
| Multi-tenant SaaS | Single platform instance, self-hosted |

---

## 4. User Roles & Stories

### 4.1 Participant

> **As a participant, I want to...**

#### Authentication
- **P-AUTH-1**: Register an account with email and password so I can access the platform.
  - *Acceptance*: POST /auth/register → 201, verification email sent (or in-app token)
  - *Invariant*: Email must be unique (DB unique constraint)

- **P-AUTH-2**: Verify my email address before I can participate.
  - *Acceptance*: GET /auth/verify?token=... → account marked verified, JWT issued

- **P-AUTH-3**: Log in with email and password to receive a JWT session token.
  - *Acceptance*: POST /auth/login → 200 + JWT (24hr expiry, refreshable)

- **P-AUTH-4**: Log out and invalidate my session.
  - *Acceptance*: POST /auth/logout → token blacklisted / refresh revoked

#### Event Registration
- **P-REG-1**: Browse the list of public events on the platform.
  - *Acceptance*: GET /events → list of events with status, dates, track names

- **P-REG-2**: Register for an event during the registration window.
  - *Acceptance*: POST /events/{slug}/register → 201 EventRegistration created, participant role granted

- **P-REG-3**: Un-register from an event before I have submitted a project.
  - *Acceptance*: DELETE /events/{slug}/register → 200 if no submitted submission, 409 if submitted (I19)

#### Team Formation
- **P-TEAM-1**: Create a new team for an event and receive an invite code.
  - *Acceptance*: POST /events/{slug}/teams → 201 + invite_code; user becomes owner
  - *Invariant*: User can only be on one team per event (I1)

- **P-TEAM-2**: Join an existing team using an invite code.
  - *Acceptance*: POST /teams/join { invite_code } → 200; user added as member
  - *Invariant*: Team not at max_team_size; user not already on a team (I1); team not locked

- **P-TEAM-3**: View my team's current members and invite code.
  - *Acceptance*: GET /teams/{id} → team details, member list, invite_code (owner only)

- **P-TEAM-4**: Leave a team before the submission deadline (non-owners only).
  - *Acceptance*: DELETE /teams/{id}/members/me → 200 if team not locked, 409 if locked

#### Project Submission
- **P-SUB-1**: Create a draft submission during the submission window.
  - *Acceptance*: POST /events/{slug}/submissions → 201, status=draft; team locked (I2: one submission per team)

- **P-SUB-2**: Edit my draft submission's title, description, URLs, and track before the deadline.
  - *Acceptance*: PATCH /submissions/{id} → 200; rejected with 403 after deadline (I4)

- **P-SUB-3**: Upload a cover image for my submission.
  - *Acceptance*: POST /submissions/{id}/cover → stores file locally, returns cover_image_url

- **P-SUB-4**: Submit my final project before the deadline.
  - *Acceptance*: POST /submissions/{id}/submit → 200, status=submitted, submitted_at set
  - *Acceptance*: Rejected with 403 after submission_deadline_at (I4)
  - *Acceptance*: Required fields validated: title, track_id, at least one of demo_url or repo_url

- **P-SUB-5**: View my submission after submitting to confirm it was recorded.
  - *Acceptance*: GET /submissions/{id} → full submission detail for team members

#### Gallery & Results
- **P-GAL-1**: Browse the public gallery of submitted projects.
  - *Acceptance*: GET /events/{slug}/gallery → list of submitted projects, searchable by title/tag
  - *Acceptance*: Draft and disqualified submissions not shown (I11)
  - *Acceptance*: Scores hidden until results_published (I9 analog for gallery)

- **P-GAL-2**: Search and filter the gallery by track or keyword.
  - *Acceptance*: GET /events/{slug}/gallery?track=ai&q=react → filtered results

- **P-GAL-3**: View the final results and rankings after they are published.
  - *Acceptance*: GET /events/{slug}/results → ranked submissions with final_score, per track

- **P-CERT-1** *(T4)*: Download my participation or winner certificate.
  - *Acceptance*: GET /certificates/{id} → certificate page with verification hash

---

### 4.2 Judge

> **As a judge, I want to...**

#### Onboarding
- **J-ON-1**: Accept a judge invitation and create my account (or link to existing).
  - *Acceptance*: Invite link with token → account created + judge role assigned for event

- **J-ON-2**: See an overview of the event I'm judging (timeline, rubric, number of assignments).
  - *Acceptance*: GET /judge/dashboard → event info, rubric criteria, assignment stats

#### Evaluation
- **J-EVAL-1**: See a queue of all submissions assigned to me.
  - *Acceptance*: GET /judge/assignments → list with status (pending, in_progress, completed)

- **J-EVAL-2**: Open a submission and read the full project detail.
  - *Acceptance*: GET /judge/assignments/{id} → submission detail + rubric criteria + my existing scores
  - *Invariant*: Only my own prior scores are shown — never other judges' (I9)
  - *Invariant*: Participants cannot access this endpoint (I17)

- **J-EVAL-3**: Score each rubric criterion with a numeric score and optional comment.
  - *Acceptance*: POST /judge/assignments/{id}/scores { criterion_id, raw_score, comment }
  - *Invariant*: raw_score must be 0 ≤ score ≤ criterion.max_score
  - *Invariant*: Rejected after judging_deadline_at (I6)

- **J-EVAL-4**: Save my scores partially and return later to complete the evaluation.
  - *Acceptance*: Partial scores are saved (UPSERT); assignment status = in_progress until all criteria scored

- **J-EVAL-5**: Submit my complete evaluation and mark the assignment as done.
  - *Acceptance*: POST /judge/assignments/{id}/complete → 200, status=completed, completed_at set
  - *Acceptance*: Rejected if any criterion has no score

- **J-EVAL-6**: Declare a conflict of interest and recuse myself from an assignment.
  - *Acceptance*: POST /judge/assignments/{id}/recuse → status=recused, organizer notified

#### Progress
- **J-PROG-1**: See how many assignments I have completed vs. remaining.
  - *Acceptance*: GET /judge/dashboard → { total, completed, in_progress, pending, recused }

---

### 4.3 Organizer

> **As an organizer, I want to...**

#### Event Configuration
- **O-EVT-1**: Create a new hackathon event in draft state.
  - *Acceptance*: POST /events → 201, status=draft; required fields: title, slug, all date windows

- **O-EVT-2**: Configure event date windows (registration, submission, judging, voting).
  - *Acceptance*: PATCH /events/{id} → updates date fields; validates ordering (I15)

- **O-EVT-3**: Add multiple tracks to an event with prize descriptions.
  - *Acceptance*: POST /events/{id}/tracks → 201 Track created with name, description, prizes (jsonb)

- **O-EVT-4**: Create a scoring rubric with weighted criteria.
  - *Acceptance*: POST /events/{id}/rubrics → 201 Rubric; POST /rubrics/{id}/criteria → 201 Criterion
  - *Invariant*: Criterion weights must sum to 1.0 to mark rubric complete (I14)

- **O-EVT-5**: Publish the event so participants can register.
  - *Acceptance*: POST /events/{id}/publish → status=registration_open
  - *Acceptance*: Blocked if: no tracks, no complete rubric, missing date fields (I15)

- **O-EVT-6**: Manually advance event status when needed.
  - *Acceptance*: POST /events/{id}/status → allowed transitions only (I15)

#### Judge Management
- **O-JDG-1**: Invite judges to the event by email.
  - *Acceptance*: POST /events/{id}/judges/invite { email } → invite created, email or in-app notification sent

- **O-JDG-2**: View all invited and active judges for the event.
  - *Acceptance*: GET /events/{id}/judges → list with status (invited, active, assignment_count)

- **O-JDG-3**: Manually assign a judge to a specific submission.
  - *Acceptance*: POST /events/{id}/assignments { judge_id, submission_id }
  - *Invariant*: Rejected if judge is on submission's team (I3)

- **O-JDG-4**: Trigger algorithmic batch assignment that covers all submissions.
  - *Acceptance*: POST /events/{id}/assignments/auto → creates assignments; each submission gets N judges (event.judges_per_submission)
  - *Invariant*: No judge assigned to own team's submission (I3)
  - *Acceptance*: Returns summary { submissions_covered, assignments_created, judges_used }

- **O-JDG-5**: View real-time judge progress across all assignments.
  - *Acceptance*: GET /events/{id}/judging-progress → per-judge and per-submission completion stats

- **O-JDG-6**: Re-assign a recused judge's submission to a replacement.
  - *Acceptance*: POST /events/{id}/assignments { judge_id: new_judge, submission_id } → creates new assignment

#### Scoring & Results
- **O-SCORE-1**: View all raw scores across all judges and submissions.
  - *Acceptance*: GET /events/{id}/scores → all Score rows with judge names, raw scores, criteria
  - *Acceptance*: Organizer can see all scores; judges can only see their own (I9)

- **O-SCORE-2**: Trigger score normalization after judging is complete using Z-score per-judge normalization.
  - *Acceptance*: POST /events/{id}/normalize → 202 (async job); blocked if judging incomplete and deadline not passed (I8)
  - *Acceptance*: Re-triggerable before publication — idempotent (REL-3)

  **Normalization Algorithm (exact — must be implemented identically by both teammates):**

  **Step 1 — Per-judge Z-score normalization**
  For each judge `j`, across ALL criteria scores they have submitted for this event:
  ```
  μ_j  = MEAN  (all raw_score values submitted by judge j in this event)
  σ_j  = STDDEV(all raw_score values submitted by judge j in this event)

  For each Score row where judge_id = j:
    normalized_score = (raw_score - μ_j) / σ_j
  ```
  Edge cases:
  - If `σ_j = 0` (judge scored identically across all criteria): set `normalized_score = 0.0` and flag the judge in the normalization report
  - Only `completed` JudgeAssignment rows are included — `recused` rows are excluded
  - Raw scores are never overwritten; `normalized_score` is a separate nullable column

  **Step 2 — Weighted normalized total per (judge × submission)**
  ```
  weighted_norm_j_S = SUM(normalized_score_ij × criterion_i.weight)
                      for all criteria i scored by judge j on submission S
  ```

  **Step 3 — Final score per submission**
  ```
  final_score_S = AVG(weighted_norm_j_S)
                  across all judges j with assignment status = 'completed'
  ```

  **Step 4 — Optional voting weight blend (T3, if event.voting_weight > 0)**
  ```
  vote_score_S    = vote_count_S / MAX(vote_count across all submissions in event)
  final_score_S   = (1 - event.voting_weight) × judge_final_score_S
                  + (event.voting_weight)      × vote_score_S
  ```
  Applied only after voting window closes. If `voting_weight = 0` (default), skip this step.

  **Step 5 — Rankings within each track**
  ```
  ORDER BY final_score DESC within each track
  Tiebreaker (in order):
    1. Higher average raw weighted total (before normalization)
    2. More completed judge assignments (more coverage = more confidence)
    3. Earlier submitted_at timestamp (earlier submission wins tie)
  ```

  *Acceptance*: Normalization report written to AuditLog metadata including per-judge μ, σ, and per-submission raw vs normalized weighted total

- **O-SCORE-3**: Preview normalized results before publishing.
  - *Acceptance*: GET /events/{id}/results/preview → ranked list with final_score, rank, track, judge_count per submission; only visible to organizer
  - *Acceptance*: Shows both raw_weighted_total and normalized final_score side-by-side so organizer can see the normalization effect

- **O-SCORE-4**: Publish results to make them publicly visible.
  - *Acceptance*: POST /events/{id}/results/publish → event status=results_published
  - *Invariant*: Blocked until normalization complete (I16)

- **O-SCORE-5**: Disqualify a submission with a reason.
  - *Acceptance*: POST /submissions/{id}/disqualify { reason } → status=disqualified, excluded from gallery (I11), AuditLog written

#### Data Export
- **O-EXP-1**: Export submissions as CSV at any event stage.
  - *Acceptance*: GET /events/{id}/export/submissions.csv → id, team, track, title, status, submitted_at

- **O-EXP-2**: Export all scores (raw and normalized) as CSV.
  - *Acceptance*: GET /events/{id}/export/scores.csv → submission_id, judge_id, criterion, raw_score, normalized_score, comment

- **O-EXP-3**: Export final rankings as CSV.
  - *Acceptance*: GET /events/{id}/export/rankings.csv → rank, track, submission_title, team, final_score

- **O-EXP-4**: Export the event audit log as CSV.
  - *Acceptance*: GET /events/{id}/export/audit.csv → actor, action, target, timestamp, metadata

#### Community Voting (T3)
- **O-VOTE-1**: Enable and configure community voting for an event.
  - *Acceptance*: PATCH /events/{id} { voting_opens_at, voting_closes_at, voting_weight } → voting configured

- **O-VOTE-2**: View voting statistics and flagged suspicious accounts.
  - *Acceptance*: GET /events/{id}/voting-stats → vote counts per submission, IP hash clusters, flagged accounts

#### Certificates (T4)
- **O-CERT-1**: Generate certificates for all participants, winners, and judges after results publish.
  - *Acceptance*: POST /events/{id}/certificates/generate → batch creates Certificate records with HMAC hashes

---

### 4.4 Admin

> **As an admin, I want to...**

- **A-ADM-1**: Create organizer accounts on the platform.
  - *Acceptance*: POST /admin/users { email, role: 'organizer' } → account created, invite sent

- **A-ADM-2**: View and manage all events across the platform.
  - *Acceptance*: GET /admin/events → all events with status

- **A-ADM-3**: Impersonate any user for support purposes (always logged).
  - *Acceptance*: POST /admin/impersonate { user_id } → returns JWT as that user
  - *Invariant*: AuditLog write is unconditional before token is issued (I18)

- **A-ADM-4**: Deactivate a user account.
  - *Acceptance*: PATCH /admin/users/{id} { is_active: false } → user cannot log in; existing JWTs rejected

- **A-ADM-5**: View the full platform-wide audit log.
  - *Acceptance*: GET /admin/audit-log → all AuditLog rows, filterable by action, actor, event, date range

---

## 5. Community Voting User Stories (T3)

> **As an authenticated user, I want to...**

- **V-VOTE-1**: Browse the gallery during the voting window.
  - *Acceptance*: Gallery shown in randomized order per session; vote counts hidden (I12)

- **V-VOTE-2**: Cast one vote for a submission I liked.
  - *Acceptance*: POST /events/{slug}/votes { submission_id } → 201; rejected if already voted (I5)

- **V-VOTE-3**: See which submissions I've already voted for.
  - *Acceptance*: GET /events/{slug}/votes/mine → list of submission_ids I've voted for

- **V-VOTE-4**: Not be able to vote more than N times per hour (rate limiting).
  - *Acceptance*: 429 with retry_after when rate limit exceeded

---

## 6. Non-Functional Requirements

### 6.1 Security
| Requirement | Detail |
|-------------|--------|
| **SEC-1** | All passwords hashed with bcrypt (cost factor ≥ 12) |
| **SEC-2** | JWT tokens expire in 24 hours; refresh tokens in 7 days |
| **SEC-3** | All endpoints require authentication except: public gallery, public event list, certificate verify page |
| **SEC-4** | Role enforcement at API middleware layer (not just UI) — invariants I10, I17 |
| **SEC-5** | AuditLog is append-only at DB privilege level (I7) |
| **SEC-6** | IP addresses are hashed before storage (SHA-256) — never stored raw |
| **SEC-7** | HMAC-SHA256 verification for certificates (I from F7) |
| **SEC-8** | Rate limiting on voting and auth endpoints (login, register, vote) |
| **SEC-9** | SQL injection prevention via parameterized queries / ORM |
| **SEC-10** | File upload validation: type, size limits, stored on local volume (not DB) |

### 6.2 Performance
| Requirement | Detail |
|-------------|--------|
| **PERF-1** | Gallery page loads in < 2s for up to 500 submissions |
| **PERF-2** | Score normalization job completes in < 30s for 500 submissions × 3 judges × 4 criteria |
| **PERF-3** | API p99 response time < 500ms under normal load |
| **PERF-4** | Platform supports 500 concurrent users during voting window |
| **PERF-5** | Pagination on all list endpoints (default 20, max 100 per page) |

### 6.3 Reliability
| Requirement | Detail |
|-------------|--------|
| **REL-1** | Database migrations run automatically on startup |
| **REL-2** | Seed data loads idempotently on first run (re-running `docker compose up` is safe) |
| **REL-3** | Normalization job is idempotent — can be re-run safely before publication |
| **REL-4** | All state transitions are transactional — no partial writes |
| **REL-5** | Health check endpoint: GET /health → 200 { db: ok, app: ok } |

### 6.4 Operability
| Requirement | Detail |
|-------------|--------|
| **OPS-1** | Single command start: `docker compose up` |
| **OPS-2** | No dependencies on hosted services, cloud accounts, or external APIs |
| **OPS-3** | Works with network completely disabled after initial image pull |
| **OPS-4** | `.env.example` documents all required environment variables |
| **OPS-5** | README documents: prerequisites, how to run, how to run tests, how to seed, how to reset |
| **OPS-6** | Structured JSON logs to stdout (parseable) |
| **OPS-7** | Graceful shutdown on SIGTERM (in-flight requests complete) |

### 6.5 Observability
| Requirement | Detail |
|-------------|--------|
| **OBS-1** | Structured logs for every request: method, path, status, duration, user_id |
| **OBS-2** | AuditLog covers all state-changing operations (see AuditLog.action enum) |
| **OBS-3** | Health endpoint exposes DB connection status |

### 6.6 Developer Experience
| Requirement | Detail |
|-------------|--------|
| **DX-1** | Test suite runs with a single command |
| **DX-2** | Database migrations are numbered and reversible |
| **DX-3** | Seed script creates: 1 admin, 2 organizers, 10 judges, 30 participants, 1 event with 3 tracks, 1 rubric with 4 criteria, 20 teams, 20 submissions, full judge assignments, scores |

---

## 7. Tier Completion Criteria

### T1 — Core (Required to be judged)
- [ ] Email/password authentication with JWT
- [ ] Email verification flow
- [ ] Four distinct roles: participant, judge, organizer, admin
- [ ] Event creation with all date windows
- [ ] Track and prize configuration
- [ ] Participant event registration and un-registration
- [ ] Team creation with invite codes
- [ ] Team joining via invite code
- [ ] Project submission with draft and edit functionality
- [ ] Deadline enforcement (server-side, not just UI)
- [ ] Public gallery with search and track filtering
- [ ] Gallery excludes drafts and disqualified submissions

### T2 — Judging
- [ ] Judge invitation by email
- [ ] Algorithmic batch assignment (round-robin, conflict-aware)
- [ ] Manual judge assignment
- [ ] Weighted, configurable rubrics
- [ ] Backend-enforced role isolation (I9, I10, I17)
- [ ] Per-criterion scoring with comments
- [ ] Partial score saves (in_progress state)
- [ ] Judge progress dashboard (per-judge and per-submission)
- [ ] Conflict of interest / recusal flow
- [ ] Z-score normalization per judge
- [ ] Final score aggregation and track rankings
- [ ] Results preview (organizer-only before publish)
- [ ] Results publication
- [ ] CSV exports: submissions, scores, rankings, audit log

### T3 — Public
- [ ] Community voting (one vote per user per submission)
- [ ] Vote counts hidden during voting window (I12)
- [ ] Randomized gallery order during voting
- [ ] Rate limiting on votes
- [ ] IP hash storage for Sybil detection
- [ ] Organizer voting stats dashboard with flagged accounts
- [ ] Submission comments (per project)
- [ ] AuditLog for all vote events

### T4 — Stretch
- [ ] REST API OpenAPI specification covering all UI actions
- [ ] Webhook support for key events (submission, result publish)
- [ ] Participation certificates with public verify URL
- [ ] Winner certificates with rank
- [ ] Judge service certificates
- [ ] HMAC-SHA256 tamper-proof verification
- [ ] Embeddable gallery widget (iframe or web component)
- [ ] Bulk import/export (events, participants, submissions)

---

## 8. Bonus Challenge Requirements

### Normalization Proof (+5)
- Load provided fixture data
- Run normalization
- Output report: per-judge μ, σ, raw score range; per-submission raw vs normalized weighted total
- Demonstrate ranking changes caused by normalization
- Include in `acceptance-report.txt`

### Pairwise Mode (+5)
- Allow organizer to enable "pairwise judging" mode on an event
- Judges presented with two submissions side-by-side and pick a winner
- System accumulates pairwise results and runs Bradley-Terry estimator
- Outputs rankings with confidence intervals
- Document the model in JUDGING.md

### Threat Model (+3)
- Publish `THREAT-MODEL.md`
- Cover attack vectors: Sybil voting, ballot stuffing, submission scraping, judge collusion, deadline gaming
- For each: describe the attack, the detection mechanism, and the prevention mechanism

### API First (+3)
- Every UI action is backed by a documented REST endpoint
- OpenAPI 3.0 spec in `openapi.yaml`
- Spec is auto-generated or kept in sync with implementation
- Spec served at `/api/docs` (Swagger UI or Redoc, no external CDN)

---

## 9. Required Deliverables Checklist

| Deliverable | Requirement |
|-------------|-------------|
| **Public GitHub repository** | OSI-approved license (MIT recommended) |
| **Working implementation** | Passes acceptance suite for claimed tiers |
| **`docker compose up`** | Starts seeded, working portal |
| **`acceptance-report.txt`** | Acceptance suite output, committed to repo root |
| **`README.md`** | Prerequisites, setup, run, test, reset instructions |
| **`ARCHITECTURE.md`** | System design, service topology, tech decisions |
| **`DATA-MODEL.md`** | Schema, all entities, import/export paths |
| **`JUDGING.md`** | Assignment strategy, scoring, normalization method, rationale |
| **Tests** | Unit tests for normalization math; integration tests for critical flows |
| **5-min demo video** | One complete event lifecycle from registration to results |

---

## 10. Open Questions (Pending Sep 24 Full Spec)

| # | Question | Impact |
|---|----------|--------|
| Q1 | What HTTP endpoints does the acceptance suite test? Format of requests/responses? | API shape may need to match exactly |
| Q2 | What is the fixture data format? (JSON? SQL dump?) | Seed script implementation |
| Q3 | Are rubric criterion scores integer-only or decimal? | Score validation, storage type |
| Q4 | Is there a defined minimum number of judges per submission? | Affects assignment algorithm default |
| Q5 | What constitutes a "signed" judge participation record? (HMAC? GPG?) | Certificate implementation |
| Q6 | What file types can be uploaded for cover images? Size limits? | File storage implementation |
| Q7 | What is the embeddable gallery widget format? (iframe? web component? script embed?) | T4 implementation approach |
| Q8 | Does voting_weight apply globally or can it differ per track? | Data model decision |
| Q9 | Are there any specific OpenAPI conventions the acceptance suite validates? | API spec format |
| Q10 | Is there a specific format for the acceptance-report.txt? | Reporting output format |

*All open questions to be resolved on September 24 when the full specification is published.*
*Update MASTER-CONTEXT.md and this PRD accordingly before hackathon kickoff.*

---

## 11. Acceptance Criteria Summary

A submission is acceptable for judging if and only if:

1. `docker compose up` starts successfully from a clean state
2. Platform is fully functional with the network disabled
3. Fixture data is present and queryable after startup
4. `acceptance-report.txt` is present in repo root
5. T1 features work as described in section 7
6. All invariants hold under direct API testing (no UI bypass)
7. Required documents (README, ARCHITECTURE, DATA-MODEL, JUDGING) are present and substantive

---

*Next document: `DATA-MODEL.md` — full schema, indexes, and import/export paths*
*Written after consulting MASTER-CONTEXT.md*
