---
id: E-004
title: Invite Judges + List Judges — Judge Role Management
epic: events
owner: Sujay (backend)
status: "[ ] not-started"
branch: story/E-004-judges
blocks: J-001
blocked-by: E-001
---

# E-004 · Invite Judges + List Judges

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules
- `MASTER-CONTEXT.md` — Invariant **I10**: only admins/organizers can assign judge roles (enforced in use case)
- `docs/api-design.md §Events` — POST /events/{slug}/judges, GET /events/{slug}/judges
- `docs/data-model.md §event_roles` — role = 'judge', invited_by column
- `docs/invariants.md §I10` — only organizer or admin can invite judges

## What We're Building

Event organizers invite users to be judges by their email address. A judge gets an `event_roles` row with `role='judge'`. Judges cannot also be participants (I10 variation). The organizer can list all current judges for an event.

**Endpoints:**
- `POST /api/v1/events/{slug}/judges` — organizer-only: invite a user as a judge
- `DELETE /api/v1/events/{slug}/judges/{userId}` — organizer-only: remove a judge
- `GET /api/v1/events/{slug}/judges` — authenticated: list judges for this event

### POST /events/{slug}/judges
**Request:** `{ "email": "judge@example.com" }`  
**Success (201):**
```json
{
  "data": {
    "userId": "uuid", "email": "judge@example.com", "displayName": "Judge Name",
    "role": "judge", "invitedAt": "2026-10-01T10:00:00Z"
  },
  "meta": {...}
}
```
**Error cases:**
| Condition | HTTP | Code |
|-----------|------|------|
| Caller is not organizer | 403 | `FORBIDDEN` |
| Email not registered | 404 | `NOT_FOUND` |
| User already has a role (participant or judge) in this event | 409 | `DUPLICATE_RESOURCE` |
| User is the organizer themselves | 422 | `INVARIANT_VIOLATION` |

### GET /events/{slug}/judges
**Response (200):**
```json
{
  "data": [
    { "userId": "uuid", "email": "...", "displayName": "...", "role": "judge", "invitedAt": "..." }
  ],
  "meta": {...}
}
```

### DELETE /events/{slug}/judges/{userId}
**Response (200):** `{ "data": { "removed": true }, "meta": {...} }`  
**Rule:** Cannot remove a judge who has already submitted scores (enforced in use case)

## Files to Create / Modify

### internal/events/port/in.go (add)
```go
type InviteJudgeUseCase interface {
    Invite(ctx context.Context, cmd InviteJudgeCommand) (*JudgeDTO, error)
}
type RemoveJudgeUseCase interface {
    Remove(ctx context.Context, cmd RemoveJudgeCommand) error
}
type ListJudgesUseCase interface {
    List(ctx context.Context, eventSlug string, callerID uuid.UUID) ([]*JudgeDTO, error)
}

type InviteJudgeCommand struct {
    CallerID  uuid.UUID
    EventSlug string
    Email     string
}
type RemoveJudgeCommand struct {
    CallerID  uuid.UUID
    EventSlug string
    JudgeID   uuid.UUID
}
```

### internal/events/port/out.go (add)
```go
// JudgeScoreRepository (cross-module read — only the count, not actual scores)
type JudgeScoreCountRepository interface {
    HasSubmittedScores(ctx context.Context, judgeID, eventID uuid.UUID) (bool, error)
}
```

### internal/events/usecase/judges.go
```go
func (s *InviteJudgeService) Invite(ctx context.Context, cmd port.InviteJudgeCommand) (*port.JudgeDTO, error) {
    // 1. Verify caller has organizer role for this event (I10)
    // 2. Look up target user by email — 404 if not found
    // 3. Check target user != organizer — 422 if same
    // 4. Check target user has no existing role in this event — 409 if exists
    // 5. Grant judge role (event_roles INSERT with invited_by = callerID)
    // 6. Return JudgeDTO
}

func (s *RemoveJudgeService) Remove(ctx context.Context, cmd port.RemoveJudgeCommand) error {
    // 1. Verify caller is organizer (I10)
    // 2. Check judge has NOT submitted scores — 422 if they have
    // 3. Delete event_roles row
}
```

### Cross-module communication note
The `InviteJudgeService` in `events/` needs to look up a user by email. It does this via the `UserLookupPort` interface in `events/port/out.go`, which is implemented by `auth/repository`. This is the ONLY allowed cross-module call: through an interface defined in the calling module.

```go
// events/port/out.go
type UserLookup interface {
    FindByEmail(ctx context.Context, email string) (*UserSummary, error)
}
type UserSummary struct {
    ID          uuid.UUID
    Email       string
    DisplayName string
}
```
The `auth` module's `UserRepository` satisfies this interface — wired in `main.go`.

## Tests Required

### Use case tests
- `TestInviteJudgeService_ValidEmail_Returns201JudgeDTO`
- `TestInviteJudgeService_NonOrganizerCaller_Returns403` (I10)
- `TestInviteJudgeService_EmailNotFound_Returns404`
- `TestInviteJudgeService_UserAlreadyParticipant_Returns409`
- `TestInviteJudgeService_OrganizerInvitesSelf_Returns422`
- `TestRemoveJudgeService_JudgeHasScores_Returns422`

### Integration test
- Create event → invite judge → list judges → verify judge appears
- Try to re-invite same judge → 409
- Invite organizer as judge → 422

## Definition of Done
- [ ] `go test ./internal/events/...` → 100% green
- [ ] `POST /events/{slug}/judges` → 201 with judge DTO
- [ ] Non-organizer → 403 `FORBIDDEN`
- [ ] Email not found → 404
- [ ] Already has role → 409
- [ ] `GET /events/{slug}/judges` → array of judge DTOs
- [ ] `DELETE /events/{slug}/judges/{userId}` with judge who has scores → 422
