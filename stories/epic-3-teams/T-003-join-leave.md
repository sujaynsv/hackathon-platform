---
id: T-003
title: Join Team via Invite Code + Leave Team
epic: teams
owner: Keerthika (backend)
status: "[ ] not-started"
branch: story/T-003-join-leave
blocks: S-001
blocked-by: T-002
---

# T-003 · Join Team via Invite Code + Leave Team

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules
- `MASTER-CONTEXT.md` — Invariant **I1**: one team per user per event; Invariant **I4**: team size ≤ event maxTeamSize; Invariant **I8**: cannot join/leave after submissions_open
- `docs/api-design.md §Teams` — POST /teams/join, DELETE /events/{slug}/teams/leave
- `docs/data-model.md §teams, §team_members` — max_team_size from events table
- `docs/invariants.md §I1, §I4, §I8`

## What We're Building

Participants join an existing team using its invite code. The system validates: user is registered for the event, user doesn't already have a team (I1), team has room (I4 — team size < event.maxTeamSize), and it's still registration_open (I8). Leaving removes the user from the team. The team leader cannot leave (they must transfer leadership first or dissolve the team).

**Endpoints:**
- `POST /api/v1/teams/join` — join a team by invite code
- `DELETE /api/v1/events/{slug}/teams/leave` — leave current team

### POST /teams/join
**Request:** `{ "inviteCode": "HACK2026" }`  
**Success (201):**
```json
{
  "data": {
    "id": "uuid", "name": "Team Rocket", "eventId": "uuid",
    "members": [...],
    "inviteCode": "HACK2026"
  },
  "meta": {...}
}
```
**Error cases:**
| Condition | HTTP | Code |
|-----------|------|------|
| Invite code not found | 404 | `NOT_FOUND` |
| User not registered for this event | 403 | `FORBIDDEN` |
| User already on a team in this event (I1) | 409 | `DUPLICATE_RESOURCE` |
| Team is full — size == maxTeamSize (I4) | 422 | `INVARIANT_VIOLATION` |
| Event no longer in `registration_open` (I8) | 422 | `INVARIANT_VIOLATION` |

### DELETE /events/{slug}/teams/leave
**Success (200):** `{ "data": { "left": true }, "meta": {...} }`  
**Error cases:**
| Condition | HTTP | Code |
|-----------|------|------|
| User has no team | 404 | `NOT_FOUND` |
| User is the team leader | 422 | `INVARIANT_VIOLATION` |
| Event past `registration_open` (I8) | 422 | `INVARIANT_VIOLATION` |

## Files to Create / Modify

### internal/teams/domain/team.go (add)
```go
var ErrTeamFull          = errors.New("team is full")
var ErrNotRegistered     = errors.New("user is not registered for this event")
var ErrRegistrationClosed = errors.New("team changes not allowed after registration closes")
var ErrLeaderCannotLeave  = errors.New("team leader cannot leave; transfer leadership first")

// CheckCanJoin validates team join constraints.
func CheckCanJoin(eventStatus string, currentSize, maxSize int) error {
    if eventStatus != "registration_open" {
        return fmt.Errorf("%w", ErrRegistrationClosed)
    }
    if currentSize >= maxSize {
        return fmt.Errorf("%w: team has %d/%d members", ErrTeamFull, currentSize, maxSize)
    }
    return nil
}
```

### internal/teams/port/in.go (add)
```go
type JoinTeamUseCase interface {
    Join(ctx context.Context, cmd JoinTeamCommand) (*TeamDTO, error)
}
type LeaveTeamUseCase interface {
    Leave(ctx context.Context, cmd LeaveTeamCommand) error
}

type JoinTeamCommand struct {
    UserID     uuid.UUID
    InviteCode string
}
type LeaveTeamCommand struct {
    UserID    uuid.UUID
    EventSlug string
}
```

### internal/teams/usecase/join.go
```go
func (s *JoinTeamService) Join(ctx context.Context, cmd port.JoinTeamCommand) (*port.TeamDTO, error) {
    // 1. Find team by invite code — 404 if not found
    team, err := s.teams.FindByInviteCode(ctx, cmd.InviteCode)
    if err != nil {
        return nil, fmt.Errorf("%w: invite code not found", response.ErrNotFound)
    }

    // 2. Load event summary for this team's event
    event, err := s.events.FindSummaryByID(ctx, team.EventID)
    if err != nil {
        return nil, err
    }

    // 3. Check user is a registered participant
    role, _ := s.participants.GetRole(ctx, cmd.UserID, event.ID)
    if role != "participant" {
        return nil, fmt.Errorf("%w: must be a registered participant", response.ErrForbidden)
    }

    // 4. I1: check user doesn't already have a team
    existing, _ := s.teams.FindByEventAndUser(ctx, event.ID, cmd.UserID)
    if existing != nil {
        return nil, fmt.Errorf("%w: user already on a team in this event", response.ErrInvariantViolated)
    }

    // 5. I4 + I8: check team has room and event is still open
    count, _ := s.teams.MemberCount(ctx, team.ID)
    if err := domain.CheckCanJoin(event.Status, count, event.MaxTeamSize); err != nil {
        if errors.Is(err, domain.ErrTeamFull) {
            return nil, fmt.Errorf("%w: %s", response.ErrInvariantViolated, err.Error())
        }
        return nil, fmt.Errorf("%w: %s", response.ErrInvariantViolated, err.Error())
    }

    // 6. Add member
    member := &domain.TeamMember{
        UserID:   cmd.UserID,
        TeamID:   team.ID,
        EventID:  event.ID,
        Role:     "member",
        JoinedAt: time.Now().UTC(),
    }
    if err := s.teams.AddMember(ctx, member); err != nil {
        return nil, err
    }

    // 7. Reload team with members for response
    members, _ := s.teams.FindMembersByTeamID(ctx, team.ID)
    return toTeamDTO(team, members), nil
}
```

### internal/teams/usecase/leave.go
```go
func (s *LeaveTeamService) Leave(ctx context.Context, cmd port.LeaveTeamCommand) error {
    event, err := s.events.FindSummaryBySlug(ctx, cmd.EventSlug)
    if err != nil { return err }

    // I8: must be registration_open
    if event.Status != "registration_open" {
        return fmt.Errorf("%w: %s", response.ErrInvariantViolated, domain.ErrRegistrationClosed.Error())
    }

    // Find user's membership
    team, err := s.teams.FindByEventAndUser(ctx, event.ID, cmd.UserID)
    if err != nil || team == nil {
        return fmt.Errorf("%w: user has no team in this event", response.ErrNotFound)
    }

    // Check not leader
    for _, m := range team.Members {
        if m.UserID == cmd.UserID && m.Role == "leader" {
            return fmt.Errorf("%w", response.ErrInvariantViolated) // ErrLeaderCannotLeave
        }
    }

    return s.teams.RemoveMember(ctx, team.ID, cmd.UserID)
}
```

## Tests Required

### Domain tests
- `TestCheckCanJoin_OpenWithRoom_ReturnsNil`
- `TestCheckCanJoin_TeamFull_ReturnsErrTeamFull` (I4)
- `TestCheckCanJoin_EventClosed_ReturnsErrRegistrationClosed` (I8)

### Use case tests
- `TestJoinTeamService_ValidCode_ReturnsTeamDTO`
- `TestJoinTeamService_NotRegistered_Returns403`
- `TestJoinTeamService_AlreadyOnTeam_Returns409` (I1)
- `TestJoinTeamService_TeamFull_Returns422` (I4)
- `TestJoinTeamService_EventNotOpen_Returns422` (I8)
- `TestLeaveTeamService_Leader_Returns422`
- `TestLeaveTeamService_EventClosed_Returns422` (I8)
- `TestLeaveTeamService_NoTeam_Returns404`
- `TestLeaveTeamService_ValidMember_RemovesMember`

### Integration test
- Create team → share code → second user joins → 201
- Third user joins full team (maxTeamSize=2) → 422 (I4)
- Event transitions to submissions_open → try join → 422 (I8)
- Leader tries to leave → 422

## Definition of Done
- [ ] `go test ./internal/teams/...` → 100% green
- [ ] `POST /teams/join` with valid code → 201 team DTO
- [ ] Invalid code → 404
- [ ] Team full → 422 `INVARIANT_VIOLATION` (I4)
- [ ] Event not registration_open → 422 (I8)
- [ ] Already on team → 409 (I1)
- [ ] `DELETE /events/{slug}/teams/leave` by non-leader → 200
- [ ] Leader tries to leave → 422
