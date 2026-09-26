title:	[Keerthika] E-002: Update Event + State Transition
state:	OPEN
author:	sujaynsv (Sujay Nimmagadda)
labels:	
comments:	0
assignees:	
projects:	72 Hours - Stories & Tasks Board (Keerthika)
milestone:	
number:	13
--
---
id: E-002
title: Update Event + State Transition — PATCH /api/v1/events/{slug}
epic: events
owner: Keerthika (backend)
status: "[ ] not-started"
branch: story/E-002-update-status
blocks: E-003, E-004, T-001
blocked-by: E-001
---

# E-002 · Update Event + State Transition — PATCH /api/v1/events/{slug}

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules
- `MASTER-CONTEXT.md` — Invariant **I15**: only allowed state transitions (enforced in domain layer by `Event.Transition()`)
- `docs/api-design.md §Events` — PATCH /events/{slug} request/response shape
- `docs/data-model.md §events` — all updatable fields
- `docs/invariants.md §I15` — full transition table; organizer-only

## What We're Building

A PATCH endpoint that lets event organizers update event metadata AND/OR advance the event's status. Both happen in one request. The state machine is the core: status changes are validated by `Event.Transition()` in the domain layer. The endpoint is **organizer-only** — the middleware checks the caller's role from `event_roles`.

**Endpoint:** `PATCH /api/v1/events/{slug}`  
**Auth:** Required + must have role = `organizer` for this event

**Request (all fields optional — only send what you want to change):**
```json
{
  "title": "Updated Title",
  "description": "New description",
  "status": "registration_open",
  "maxTeamSize": 4,
  "registrationOpensAt": "2026-10-01T00:00:00Z",
  "registrationClosesAt": "2026-10-15T00:00:00Z",
  "submissionDeadlineAt": "2026-11-01T00:00:00Z",
  "judgingDeadlineAt": "2026-11-15T00:00:00Z",
  "votingOpensAt": "2026-11-16T00:00:00Z",
  "votingClosesAt": "2026-11-20T00:00:00Z"
}
```
**Success (200):** Updated event DTO (same shape as GET /events/{slug})

**Error cases:**
| Condition | HTTP | Code |
|-----------|------|------|
| Caller is not organizer | 403 | `FORBIDDEN` |
| Event not found | 404 | `NOT_FOUND` |
| Invalid status transition (e.g., draft → judging) | 422 | `INVALID_STATE_TRANSITION` |
| Updating a field on a locked event (e.g., changing maxTeamSize when status != draft) | 422 | `INVARIANT_VIOLATION` |

**Business rule:** Fields like `maxTeamSize`, `submissionDeadlineAt` can only be changed while status is `draft` or `registration_open`. Once the event is past that, these fields are locked.

## State Transition Table (I15 — enforced in domain)
```
draft              → registration_open
registration_open  → submissions_open
submissions_open   → judging
judging            → voting  OR  results_published
voting             → results_published
results_published  → archived
```
Any other transition (including backwards) → 422 `INVALID_STATE_TRANSITION`

## Files to Create / Modify

### internal/events/port/in.go (add)
```go
type UpdateEventUseCase interface {
    Update(ctx context.Context, cmd UpdateEventCommand) (*EventDetailDTO, error)
}

type UpdateEventCommand struct {
    CallerID             uuid.UUID
    Slug                 string
    Title                *string
    Description          *string
    NewStatus            *domain.EventStatus
    MaxTeamSize          *int
    RegistrationOpensAt  *time.Time
    RegistrationClosesAt *time.Time
    SubmissionDeadlineAt *time.Time
    JudgingDeadlineAt    *time.Time
    VotingOpensAt        *time.Time
    VotingClosesAt       *time.Time
}
```

### internal/events/usecase/update.go
```go
package usecase

type UpdateEventService struct {
    events port.EventRepository
    roles  port.EventRoleRepository
    tracks port.TrackRepository
}

func (s *UpdateEventService) Update(ctx context.Context, cmd port.UpdateEventCommand) (*port.EventDetailDTO, error) {
    // 1. Load event
    event, err := s.events.FindBySlug(ctx, cmd.Slug)
    if err != nil {
        return nil, err // ErrNotFound wrapped
    }

    // 2. Check organizer role (I10 enforced here, not in middleware)
    role, err := s.roles.GetRole(ctx, cmd.CallerID, event.ID)
    if err != nil || role != "organizer" {
        return nil, fmt.Errorf("%w: only organizers can update events", response.ErrForbidden)
    }

    // 3. Apply status transition first (domain validates I15)
    if cmd.NewStatus != nil {
        if err := event.Transition(*cmd.NewStatus); err != nil {
            return nil, err // Already wraps ErrInvalidTransition
        }
    }

    // 4. Apply field updates (lock check: some fields immutable after registration_open)
    if cmd.MaxTeamSize != nil {
        if event.Status != domain.StatusDraft && event.Status != domain.StatusRegistrationOpen {
            return nil, fmt.Errorf("%w: maxTeamSize cannot be changed after registration closes", response.ErrInvariantViolated)
        }
        event.MaxTeamSize = *cmd.MaxTeamSize
    }
    if cmd.Title != nil { event.Title = *cmd.Title }
    if cmd.Description != nil { event.Description = cmd.Description }
    // ... apply all other optional fields

    event.UpdatedAt = time.Now().UTC()

    // 5. Persist
    if err := s.events.Update(ctx, event); err != nil {
        return nil, fmt.Errorf("update event: %w", err)
    }

    // 6. Load tracks for response
    tracks, _ := s.tracks.FindByEventID(ctx, event.ID)
    return toEventDetailDTO(event, tracks, "organizer"), nil
}
```

### internal/events/handler/handler.go (add PATCH)
```go
func (h *EventHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
    slug := chi.URLParam(r, "slug")
    callerID := middleware.GetUserIDAsUUID(r.Context())

    var body updateEventRequest
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        response.BadRequest(w, r, "VALIDATION_ERROR", "invalid request body")
        return
    }

    result, err := h.update.Update(r.Context(), port.UpdateEventCommand{
        CallerID:    callerID,
        Slug:        slug,
        Title:       body.Title,
        Description: body.Description,
        NewStatus:   body.Status,
        MaxTeamSize: body.MaxTeamSize,
        // ... other fields
    })
    if err != nil {
        response.HandleDomainError(w, r, err)
        return
    }
    response.OK(w, r, result)
}
```

### internal/events/repository/postgres.go (add Update)
```go
func (r *EventRepository) Update(ctx context.Context, event *domain.Event) error {
    const q = `
        UPDATE events
        SET title=$2, description=$3, status=$4, max_team_size=$5,
            registration_opens_at=$6, registration_closes_at=$7,
            submission_deadline_at=$8, judging_deadline_at=$9,
            voting_opens_at=$10, voting_closes_at=$11, updated_at=$12
        WHERE id=$1`
    _, err := r.db.ExecContext(ctx, q,
        event.ID, event.Title, event.Description, event.Status, event.MaxTeamSize,
        event.RegistrationOpensAt, event.RegistrationClosesAt,
        event.SubmissionDeadlineAt, event.JudgingDeadlineAt,
        event.VotingOpensAt, event.VotingClosesAt, event.UpdatedAt,
    )
    return err
}
```

## Tests Required

### Domain tests
- `TestEvent_Transition_DraftToRegistrationOpen_Succeeds`
- `TestEvent_Transition_DraftToJudging_ReturnsErrInvalidTransition` (I15)
- `TestEvent_Transition_RegistrationOpenToDraft_ReturnsErrInvalidTransition` (backwards)
- `TestEvent_Transition_ThroughFullLifecycle_Succeeds` — draft→reg→sub→judging→voting→results→archived

### Use case tests
- `TestUpdateEventService_ValidTitleChange_ReturnsUpdatedDTO`
- `TestUpdateEventService_NonOrganizerCaller_ReturnsErrForbidden`
- `TestUpdateEventService_InvalidTransition_ReturnsErrInvalidTransition`
- `TestUpdateEventService_ChangeMaxTeamSizeAfterRegistrationClose_ReturnsErrInvariantViolated`

### Handler tests
- `TestUpdateEventHandler_ValidBody_Returns200`
- `TestUpdateEventHandler_Unauthenticated_Returns401`
- `TestUpdateEventHandler_NonOrganizer_Returns403`
- `TestUpdateEventHandler_InvalidTransition_Returns422_INVALID_STATE_TRANSITION`

### Integration test
- `POST /events` → create draft → `PATCH /events/{slug}` with `status: "registration_open"` → 200
- Try `PATCH /events/{slug}` with `status: "judging"` (skip) → 422

## Definition of Done
- [ ] `go test ./internal/events/...` → 100% green
- [ ] `PATCH /api/v1/events/{slug}` by organizer → 200 with updated fields
- [ ] Non-organizer caller → 403 `FORBIDDEN`
- [ ] Invalid transition (e.g., draft→judging) → 422 `INVALID_STATE_TRANSITION`
- [ ] All 7 status hops in transition table verified by tests
- [ ] Field lock (maxTeamSize after registration) → 422 `INVARIANT_VIOLATION`

