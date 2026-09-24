---
id: T-002
title: Create Team + Generate Invite Code
epic: teams
owner: Keerthika (backend)
status: "[ ] not-started"
branch: story/T-002-create-team
blocks: T-003
blocked-by: T-001
---

# T-002 · Create Team + Generate Invite Code

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules
- `MASTER-CONTEXT.md` — Invariant **I1**: one team per user per event (UNIQUE on team_members.user_id + event_id); Invariant **I6**: team can only be created during `registration_open` status
- `docs/api-design.md §Teams` — POST /events/{slug}/teams, GET /events/{slug}/teams/mine
- `docs/data-model.md §teams, §team_members` — invite_code UNIQUE, role = 'leader' for creator
- `docs/invariants.md §I1, §I6`

## What We're Building

A registered participant creates a team for an event. They become the team leader (`role='leader'` in `team_members`). A unique, human-readable invite code is generated (8 alphanumeric characters, e.g. `HACK2026`). The creator is added as the first team member. Only registered participants can create teams (not judges or unregistered users).

**Endpoints:**
- `POST /api/v1/events/{slug}/teams` — create team (registered participant only)
- `GET /api/v1/events/{slug}/teams/mine` — get my team for this event (returns 404 if no team)

### POST /events/{slug}/teams
**Request:** `{ "name": "Team Rocket" }`  
**Success (201):**
```json
{
  "data": {
    "id": "uuid",
    "eventId": "uuid",
    "name": "Team Rocket",
    "inviteCode": "HACK2026",
    "members": [
      { "userId": "uuid", "displayName": "Alice", "avatarUrl": null, "role": "leader", "joinedAt": "..." }
    ]
  },
  "meta": {...}
}
```
**Error cases:**
| Condition | HTTP | Code |
|-----------|------|------|
| User not registered as participant | 403 | `FORBIDDEN` |
| User already has a team in this event (I1) | 409 | `DUPLICATE_RESOURCE` |
| Event is not in `registration_open` status (I6) | 422 | `INVARIANT_VIOLATION` |
| Team name already taken in this event | 409 | `DUPLICATE_RESOURCE` |

### GET /events/{slug}/teams/mine
**Success (200):** Same team DTO as POST response  
**Error:** User has no team → 404 `NOT_FOUND`

## Files to Create

### internal/teams/domain/team.go
```go
package domain

import (
    "crypto/rand"
    "encoding/hex"
    "errors"
    "fmt"
    "strings"
    "time"
    "github.com/google/uuid"
)

var ErrAlreadyOnTeam    = errors.New("user is already on a team in this event")
var ErrTeamNameTaken    = errors.New("team name already taken in this event")
var ErrEventNotRegOpen  = errors.New("team creation only allowed during registration_open")
var ErrNotParticipant   = errors.New("user must be a registered participant to create a team")

type Team struct {
    ID         uuid.UUID
    EventID    uuid.UUID
    Name       string
    InviteCode string
    Members    []TeamMember
    CreatedAt  time.Time
}

type TeamMember struct {
    UserID      uuid.UUID
    TeamID      uuid.UUID
    EventID     uuid.UUID
    DisplayName string
    AvatarURL   *string
    Role        string // "leader" or "member"
    JoinedAt    time.Time
}

// GenerateInviteCode creates a random 8-character uppercase alphanumeric code.
// Collision is handled by retrying at the use case level if UNIQUE constraint is violated.
func GenerateInviteCode() (string, error) {
    b := make([]byte, 4)
    if _, err := rand.Read(b); err != nil {
        return "", fmt.Errorf("generate invite code: %w", err)
    }
    return strings.ToUpper(hex.EncodeToString(b)), nil
}
```

### internal/teams/port/in.go (add)
```go
type CreateTeamUseCase interface {
    Create(ctx context.Context, cmd CreateTeamCommand) (*TeamDTO, error)
}
type GetMyTeamUseCase interface {
    GetMyTeam(ctx context.Context, userID uuid.UUID, eventSlug string) (*TeamDTO, error)
}

type CreateTeamCommand struct {
    UserID    uuid.UUID
    EventSlug string
    TeamName  string
}
```

### internal/teams/port/out.go (add)
```go
type TeamRepository interface {
    Save(ctx context.Context, team *domain.Team) error
    FindByEventAndUser(ctx context.Context, eventID, userID uuid.UUID) (*domain.Team, error)
    ExistsByName(ctx context.Context, eventID uuid.UUID, name string) (bool, error)
    FindByInviteCode(ctx context.Context, code string) (*domain.Team, error)
    AddMember(ctx context.Context, member *domain.TeamMember) error
    MemberCount(ctx context.Context, teamID uuid.UUID) (int, error)
}
```

### internal/teams/usecase/create.go
```go
func (s *CreateTeamService) Create(ctx context.Context, cmd port.CreateTeamCommand) (*port.TeamDTO, error) {
    // 1. Load event summary
    event, err := s.events.FindSummaryBySlug(ctx, cmd.EventSlug)
    if err != nil {
        return nil, err
    }

    // 2. I6: event must be in registration_open
    if event.Status != "registration_open" {
        return nil, fmt.Errorf("%w: %s", response.ErrInvariantViolated, domain.ErrEventNotRegOpen.Error())
    }

    // 3. Check caller is a registered participant (not judge, not unregistered)
    role, _ := s.participants.GetRole(ctx, cmd.UserID, event.ID)
    if role != "participant" {
        return nil, fmt.Errorf("%w: %s", response.ErrForbidden, domain.ErrNotParticipant.Error())
    }

    // 4. I1: check user does not already have a team
    existing, _ := s.teams.FindByEventAndUser(ctx, event.ID, cmd.UserID)
    if existing != nil {
        return nil, fmt.Errorf("%w: %s", response.ErrInvariantViolated, domain.ErrAlreadyOnTeam.Error())
    }

    // 5. Check team name uniqueness
    nameTaken, _ := s.teams.ExistsByName(ctx, event.ID, cmd.TeamName)
    if nameTaken {
        return nil, fmt.Errorf("%w: team name '%s' already taken", response.ErrInvariantViolated, cmd.TeamName)
    }

    // 6. Generate invite code (retry on collision — max 3 attempts)
    var code string
    for i := 0; i < 3; i++ {
        code, err = domain.GenerateInviteCode()
        if err != nil {
            return nil, err
        }
        if existing, _ := s.teams.FindByInviteCode(ctx, code); existing == nil {
            break
        }
    }

    // 7. Create team
    now := time.Now().UTC()
    team := &domain.Team{
        ID:         uuid.New(),
        EventID:    event.ID,
        Name:       cmd.TeamName,
        InviteCode: code,
        CreatedAt:  now,
    }
    if err := s.teams.Save(ctx, team); err != nil {
        return nil, err
    }

    // 8. Add creator as leader
    leader := &domain.TeamMember{
        UserID:   cmd.UserID,
        TeamID:   team.ID,
        EventID:  event.ID,
        Role:     "leader",
        JoinedAt: now,
    }
    if err := s.teams.AddMember(ctx, leader); err != nil {
        return nil, err
    }

    return toTeamDTO(team, []domain.TeamMember{*leader}), nil
}
```

## Tests Required

### Domain tests
- `TestGenerateInviteCode_Returns8CharUpperAlphanumeric`
- `TestGenerateInviteCode_IsUniqueBetweenCalls` (statistical: generate 1000, check no collision)

### Use case tests
- `TestCreateTeamService_ValidInput_Returns201WithLeaderMember`
- `TestCreateTeamService_NotParticipant_Returns403` (not registered)
- `TestCreateTeamService_UserAlreadyHasTeam_Returns409` (I1)
- `TestCreateTeamService_EventNotRegistrationOpen_Returns422` (I6)
- `TestCreateTeamService_DuplicateTeamName_Returns409`
- `TestGetMyTeamService_NoTeam_Returns404`

### Integration test
- Register for event → create team → 201 with inviteCode
- Create second team with same user → 409 (I1)
- GET /teams/mine → 200 same team DTO
- Try to create team when event is `draft` → 422 (I6)

## Definition of Done
- [ ] `go test ./internal/teams/...` → 100% green
- [ ] `POST /events/{slug}/teams` → 201 with unique inviteCode
- [ ] User already on team → 409 `DUPLICATE_RESOURCE` (I1)
- [ ] Non-participant caller → 403 `FORBIDDEN`
- [ ] Event not registration_open → 422 `INVARIANT_VIOLATION` (I6)
- [ ] `GET /events/{slug}/teams/mine` → 200 team DTO or 404
- [ ] InviteCode is 8 chars, uppercase alphanumeric, UNIQUE in DB
