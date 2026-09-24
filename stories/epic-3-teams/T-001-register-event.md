---
id: T-001
title: Register for Event + Unregister — Participant Registration
epic: teams
owner: Sujay (backend)
status: "[ ] not-started"
branch: story/T-001-event-registration
blocks: T-002
blocked-by: E-001
---

# T-001 · Register for Event + Unregister

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules
- `MASTER-CONTEXT.md` — Invariant **I3**: cannot register if event status != `registration_open`; Invariant **I19**: cannot unregister if user already has a team (team is locked)
- `docs/api-design.md §Teams` — POST /events/{slug}/register, DELETE /events/{slug}/register
- `docs/data-model.md §event_roles` — role = 'participant'; one row per user per event
- `docs/invariants.md §I3, §I19` — registration window check + team-locked check

## What We're Building

Any authenticated user can register to participate in an event. Registration creates an `event_roles` row with `role='participant'`. Registration is only allowed when the event's status is `registration_open` AND the current time is within the registration window (I3). Unregistration removes the row, but only if the user has not yet joined or created a team (I19).

**Endpoints:**
- `POST /api/v1/events/{slug}/register` — authenticated: register current user as participant
- `DELETE /api/v1/events/{slug}/register` — authenticated: unregister current user

### POST /events/{slug}/register
**Request:** No body needed (user ID from JWT)  
**Success (201):**
```json
{ "data": { "registered": true, "eventId": "uuid", "eventSlug": "hackathon-2026", "registeredAt": "..." }, "meta": {...} }
```
**Error cases:**
| Condition | HTTP | Code |
|-----------|------|------|
| Event status != `registration_open` | 422 | `INVARIANT_VIOLATION` |
| Current time after `registrationClosesAt` (I3) | 422 | `DEADLINE_PASSED` |
| User already registered | 409 | `DUPLICATE_RESOURCE` |
| User is already a judge for this event | 422 | `INVARIANT_VIOLATION` |

### DELETE /events/{slug}/register
**Success (200):** `{ "data": { "unregistered": true }, "meta": {...} }`  
**Error cases:**
| Condition | HTTP | Code |
|-----------|------|------|
| User not registered | 404 | `NOT_FOUND` |
| User has a team (I19) | 422 | `INVARIANT_VIOLATION` |

## Files to Create

### internal/teams/domain/registration.go
```go
package domain

import (
    "errors"
    "fmt"
    "time"
)

var ErrEventNotOpen     = errors.New("event is not open for registration")
var ErrDeadlinePassed   = errors.New("registration deadline has passed")
var ErrAlreadyHasTeam   = errors.New("cannot unregister: user already has a team in this event")

// CheckCanRegister validates that registration is currently allowed.
// Called from the use case — pure logic, no DB.
func CheckCanRegister(eventStatus string, registrationClosesAt *time.Time) error {
    if eventStatus != "registration_open" {
        return fmt.Errorf("%w: event status is '%s'", ErrEventNotOpen, eventStatus)
    }
    if registrationClosesAt != nil && time.Now().UTC().After(*registrationClosesAt) {
        return fmt.Errorf("%w: registration closed at %s", ErrDeadlinePassed, registrationClosesAt.Format(time.RFC3339))
    }
    return nil
}
```

### internal/teams/port/in.go
```go
package port

import (
    "context"
    "github.com/google/uuid"
)

type RegisterForEventUseCase interface {
    Register(ctx context.Context, cmd RegisterCommand) (*RegistrationDTO, error)
}
type UnregisterFromEventUseCase interface {
    Unregister(ctx context.Context, cmd UnregisterCommand) error
}

type RegisterCommand struct {
    UserID    uuid.UUID
    EventSlug string
}
type UnregisterCommand struct {
    UserID    uuid.UUID
    EventSlug string
}

type RegistrationDTO struct {
    Registered   bool   `json:"registered"`
    EventID      string `json:"eventId"`
    EventSlug    string `json:"eventSlug"`
    RegisteredAt string `json:"registeredAt"`
}
```

### internal/teams/port/out.go
```go
package port

import (
    "context"
    "time"
    "github.com/google/uuid"
)

// EventSummary is what the teams module needs to know about an event.
// The events module's EventRepository satisfies this through an interface adapter in main.go.
type EventSummary struct {
    ID                   uuid.UUID
    Slug                 string
    Status               string
    RegistrationClosesAt *time.Time
}

type EventReader interface {
    FindSummaryBySlug(ctx context.Context, slug string) (*EventSummary, error)
}

type ParticipantRepository interface {
    GetRole(ctx context.Context, userID, eventID uuid.UUID) (string, error) // returns "" if not registered
    GrantParticipantRole(ctx context.Context, userID, eventID uuid.UUID) error
    RevokeParticipantRole(ctx context.Context, userID, eventID uuid.UUID) error
}

type TeamMemberRepository interface {
    HasTeamInEvent(ctx context.Context, userID, eventID uuid.UUID) (bool, error)
}
```

### internal/teams/usecase/register.go
```go
package usecase

import (
    "context"
    "fmt"
    "time"

    "github.com/dogfood/platform/internal/shared/response"
    "github.com/dogfood/platform/internal/teams/domain"
    "github.com/dogfood/platform/internal/teams/port"
)

type RegisterService struct {
    events       port.EventReader
    participants port.ParticipantRepository
}

func (s *RegisterService) Register(ctx context.Context, cmd port.RegisterCommand) (*port.RegistrationDTO, error) {
    // 1. Load event summary
    event, err := s.events.FindSummaryBySlug(ctx, cmd.EventSlug)
    if err != nil {
        return nil, err
    }

    // 2. Check I3: registration open + deadline not passed
    if err := domain.CheckCanRegister(event.Status, event.RegistrationClosesAt); err != nil {
        if errors.Is(err, domain.ErrDeadlinePassed) {
            return nil, fmt.Errorf("%w: %s", response.ErrDeadlinePassed, err.Error())
        }
        return nil, fmt.Errorf("%w: %s", response.ErrInvariantViolated, err.Error())
    }

    // 3. Check no existing role (participant, judge, organizer)
    existingRole, _ := s.participants.GetRole(ctx, cmd.UserID, event.ID)
    if existingRole == "judge" || existingRole == "organizer" {
        return nil, fmt.Errorf("%w: user already has role '%s' in this event", response.ErrInvariantViolated, existingRole)
    }
    if existingRole == "participant" {
        return nil, fmt.Errorf("%w: already registered", response.ErrInvariantViolated)
    }

    // 4. Grant participant role
    now := time.Now().UTC()
    if err := s.participants.GrantParticipantRole(ctx, cmd.UserID, event.ID); err != nil {
        return nil, err
    }

    return &port.RegistrationDTO{
        Registered:   true,
        EventID:      event.ID.String(),
        EventSlug:    event.Slug,
        RegisteredAt: now.Format(time.RFC3339),
    }, nil
}

type UnregisterService struct {
    events      port.EventReader
    participants port.ParticipantRepository
    teamMembers  port.TeamMemberRepository
}

func (s *UnregisterService) Unregister(ctx context.Context, cmd port.UnregisterCommand) error {
    // 1. Load event
    event, err := s.events.FindSummaryBySlug(ctx, cmd.EventSlug)
    if err != nil {
        return err
    }

    // 2. Verify user is registered
    role, _ := s.participants.GetRole(ctx, cmd.UserID, event.ID)
    if role != "participant" {
        return fmt.Errorf("%w: user is not registered for this event", response.ErrNotFound)
    }

    // 3. Check I19: user has no team
    hasTeam, err := s.teamMembers.HasTeamInEvent(ctx, cmd.UserID, event.ID)
    if err != nil {
        return err
    }
    if hasTeam {
        return fmt.Errorf("%w: %s", response.ErrInvariantViolated, domain.ErrAlreadyHasTeam.Error())
    }

    // 4. Remove participant role
    return s.participants.RevokeParticipantRole(ctx, cmd.UserID, event.ID)
}
```

## Tests Required

### Domain tests
- `TestCheckCanRegister_OpenEvent_WithinWindow_ReturnsNil`
- `TestCheckCanRegister_EventNotOpen_ReturnsErrEventNotOpen` (I3)
- `TestCheckCanRegister_PastDeadline_ReturnsErrDeadlinePassed` (I3)

### Use case tests
- `TestRegisterService_ValidRegistration_Returns201DTO`
- `TestRegisterService_EventNotOpen_Returns422` (I3)
- `TestRegisterService_DeadlinePassed_Returns422_DEADLINE_PASSED` (I3)
- `TestRegisterService_AlreadyRegistered_Returns409`
- `TestRegisterService_UserIsJudge_Returns422`
- `TestUnregisterService_UserHasTeam_Returns422` (I19)
- `TestUnregisterService_UserNotRegistered_Returns404`
- `TestUnregisterService_NoTeam_RemovesRole_Returns200`

### Integration test
- Register for `registration_open` event → 201
- Re-register → 409
- Try to register for `draft` event → 422
- Unregister with no team → 200
- Create team → try unregister → 422 (I19)

## Definition of Done
- [ ] `go test ./internal/teams/...` → 100% green
- [ ] `POST /events/{slug}/register` → 201 when event is open
- [ ] Status != `registration_open` → 422 `INVARIANT_VIOLATION`
- [ ] Past deadline → 422 `DEADLINE_PASSED`
- [ ] Already registered → 409 `DUPLICATE_RESOURCE`
- [ ] Unregister with team → 422 `INVARIANT_VIOLATION` (I19)
- [ ] Unregister without team → 200
