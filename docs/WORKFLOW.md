# AI SDLC Workflow — Dogfood Hackathon Platform
# How we build this project: story by story, agent-assisted, quality-gated.

---

## The Loop (One Story at a Time)

```
1. Pick the next story from stories/README.md (follow dependency order)
2. Open Antigravity in this workspace
3. Paste the FULL STORY PROMPT (see template below)
4. AI reads all referenced docs + the story → implements all layers
5. Run: go test ./... (must be 100% green)
6. Run: docker compose up → test the endpoint manually
7. git commit -m "story(STORY-ID): description"
8. git push → automated CI runs quality gates
9. If all gates green: merge to main
10. Update stories/README.md: change [ ] to [x] for this story
11. Pick next story
```

---

## The Full Story Prompt Template

Copy this EXACTLY when starting a new story session. Fill in the [ ] sections.

```
You are implementing a story for the Dogfood Hackathon Platform.
The backend is Go 1.23 + Chi v5. The frontend is Next.js 14. DO NOT use Java or Spring.

MANDATORY: Read these files BEFORE writing any code:
1. .agents/rules/engineering-standards.md   ← Go architecture rules, layer guide, code examples
2. MASTER-CONTEXT.md                         ← project rules, 19 invariants list
3. docs/api-design.md                        ← [PASTE RELEVANT ENDPOINT SECTION TITLE]
4. docs/data-model.md                        ← [PASTE RELEVANT TABLE NAMES]
5. docs/invariants.md                        ← [PASTE RELEVANT INVARIANT NUMBERS: e.g., I1, I3]
6. docs/architecture.md §2.2                 ← Go package structure
7. docs/tech-stack.md                        ← Go libraries, versions, and usage patterns

STORY: [PASTE FULL STORY CARD CONTENT FROM stories/ DIRECTORY]

IMPLEMENTATION REQUIREMENTS:
- Follow the EXACT module structure from engineering-standards.md §2
- domain/: pure Go structs and error sentinels — ZERO external imports except stdlib + uuid
- port/in.go: use case interfaces + command structs — no HTTP, no SQL
- port/out.go: repository + service interfaces — no HTTP, no SQL
- usecase/: orchestration only, calls ports — NEVER import repository/ or handler/
- handler/: HTTP only — calls port/in.go interfaces, uses response.ApiResponse[T] envelope
- repository/: SQL only — implements port/out.go, uses sqlx/pgx
- Response shapes must EXACTLY match docs/api-design.md (field names, types, nesting)
- Error codes must EXACTLY match the error code table in docs/api-design.md
- All domain errors mapped via shared/response.HandleDomainError()

TESTS REQUIRED:
- Unit test for domain logic (plain go test, zero external deps, runs < 50ms)
- Use case test: mock the port/out.go interfaces with a struct — no DB, no Redis
- Handler test: httptest.NewRecorder() — mock the port/in.go interface — no DB
- Integration test: testcontainers-go with real PostgreSQL 16
- One test per invariant listed in the story, named: TestXxx_Yyy_ReturnsZzz
- Edge cases: invalid input → 400, unauthorized → 401, state violations → 422

DEFINITION OF DONE:
- go test ./... → 100% green (zero skipped)
- go vet ./... → zero warnings
- go build ./... → no compile errors
- Endpoint returns EXACT response shape from api-design.md
- All invariants have a named test
- docker compose up → /health returns {"status":"up"}
```

---

## Parallel Frontend + Backend Strategy

Frontend and backend run in parallel. Frontend always builds against the TypeScript
types in `web/src/types/api.ts` which mirror `docs/api-design.md` exactly.

```
BACKEND STORY          FRONTEND STORY       Can Start When
─────────────────────  ───────────────────  ─────────────────────────────
A-001 (register)   →   FE-A-001 (register)  Backend A-001 merged to main
A-002 (login)      →   FE-A-001 (login)     Backend A-002 merged to main
E-001 (events)     →   FE-E-001 (list)      Backend E-001 merged to main
T-002 (teams)      →   FE-T-001 (teams)     Backend T-002 merged to main
S-001 (submission) →   FE-S-001 (submit)    Backend S-001 merged to main
J-001 (judge q)    →   FE-J-001 (judge ui)  Backend J-001 merged to main
V-001 (vote)       →   FE-V-001 (vote btn)  Backend V-001 merged to main
```

Frontend connects to the real backend via `http://localhost/api/v1` (nginx proxied).
No mocking needed — both run in the same docker-compose network.

---

## Git Workflow

```bash
# Start a story
git checkout main && git pull
git checkout -b story/[STORY-ID]-short-description
# e.g.: git checkout -b story/A-001-register-user

# During work: commit often
git add . && git commit -m "wip: domain entity + port interface"

# When done: final commit
git add . && git commit -m "story(A-001): POST /auth/register with bcrypt + I1 enforcement"

# Push and merge
git push origin story/A-001-register-user
# → CI runs → if green → merge to main (squash merge)
git checkout main && git pull
```

**Branch naming:** `story/{STORY-ID}-short-description`
**Commit format:** `story(STORY-ID): what was done`
**Merge strategy:** Squash merge to main (clean history)

---

## GitHub Project Board Setup

### Create in GitHub:
1. Go to your repo → Projects → New Project → Board view
2. Create these columns:
   - **Backlog** — all stories not yet started
   - **In Progress** — story currently being implemented
   - **Quality Gate** — `go test ./...` running / CI running
   - **Done** — merged to main

### Quality Gate Labels to create:
```
epic:foundation    (color: #6B7280)
epic:auth          (color: #8B5CF6)
epic:events        (color: #3B82F6)
epic:teams         (color: #10B981)
epic:submissions   (color: #F59E0B)
epic:judging       (color: #EF4444)
epic:voting        (color: #EC4899)
epic:admin         (color: #6366F1)
frontend           (color: #0EA5E9)
backend            (color: #84CC16)
person-a           (color: #F97316)
person-b           (color: #A855F7)
```

### Issue Template (create in .github/ISSUE_TEMPLATE/story.md):
```markdown
---
name: Story
about: Implementation story from stories/ folder
labels: ''
assignees: ''
---

## Story ID
<!-- e.g. A-001 -->

## What
<!-- One sentence description -->

## Reference Docs
- [ ] `docs/api-design.md` → [section]
- [ ] `docs/data-model.md` → [tables]
- [ ] `docs/invariants.md` → [I-numbers]

## Deliverables
<!-- List from STORIES.md -->

## Tests Required
<!-- List from STORIES.md -->

## Definition of Done
- [ ] `go test ./...` 100% green
- [ ] `go-arch-lint` passes (no dependency rule violations)
- [ ] Endpoint matches api-design.md exactly
- [ ] All invariants tested
- [ ] `docker compose up` healthy
```

---

## Definition of Done (Automated — No Peer Review)

```
A story merges to main when ALL of these are true:

QUALITY GATES:
  ✅ go test ./...    — 100% green, zero skipped
  ✅ go vet ./...     — zero warnings
  ✅ go build ./...   — compiles clean (circular imports = compile error = auto-enforced)
  ✅ go-arch-lint     — no layer violation (handler→repository forbidden, domain→pgx forbidden)
  ✅ Integration test — endpoint shape matches api-design.md exactly (testcontainers-go)
  ✅ Each invariant for this story has a named test: TestXxx_Yyy_ReturnsZzz
  ✅ Edge cases: invalid input → 400, unauthorized → 401/403, state violations → 422

STRUCTURAL CHECKS (checked by Go compiler + go-arch-lint automatically):
  ✅ domain/ has zero non-stdlib imports (except github.com/google/uuid)
  ✅ usecase/ never imports handler/ or repository/ concrete types
  ✅ handler/ only calls port/in.go interfaces — never concrete use case structs
  ✅ No cross-module internal imports (handler/ in module A never imports repository/ in module B)

NOT REQUIRED:
  ❌ Peer code review (automated quality gates replace this)
  ❌ In-memory SQLite tests (always real PostgreSQL via testcontainers-go)
  ❌ Perfect UI polish before backend story merges
```

---

## Story Status Tracking in STORIES.md

Update STORIES.md as you go:

```markdown
- [ ] A-001  Not started
- [/] A-002  In progress (on branch story/A-002-login)
- [x] A-003  Done (merged to main, commit abc1234)
```

---

## Current Status

```
Phase: DOCUMENTATION COMPLETE — READY TO BUILD
Next: F-001 (Foundation scaffold)

[ ] Epic 0 — Foundation      (F-001 → F-005)
[ ] Epic 1 — Auth            (A-001 → A-003 + FE-A-001)
[ ] Epic 2 — Events          (E-001 → E-004 + FE-E-001/002)
[ ] Epic 3 — Teams           (T-001 → T-003 + FE-T-001)
[ ] Epic 4 — Submissions     (S-001 → S-004 + FE-S-001)
[ ] Epic 5 — Judging         (J-001 → J-004 + FE-J-001)
[ ] Epic 6 — Voting          (V-001 → V-002 + FE-V-001)
[ ] Epic 7 — Admin           (ADM-001 → ADM-003 + FE-ADM-001)
```
