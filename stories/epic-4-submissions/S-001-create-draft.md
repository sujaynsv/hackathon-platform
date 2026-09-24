---
id: S-001
title: Create Submission Draft — POST /api/v1/events/{slug}/submissions
epic: submissions
owner: Sujay (backend)
status: "[ ] not-started"
branch: story/S-001-create-draft
blocks: S-002
blocked-by: T-002
---

# S-001 · Create Submission Draft

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules
- `MASTER-CONTEXT.md` — Invariant **I2**: one submission per team per event (UNIQUE on team_id + event_id); Invariant **I9**: submission only allowed during `submissions_open` status
- `docs/api-design.md §Submissions` — POST /events/{slug}/submissions request/response shape
- `docs/data-model.md §submissions` — `status = 'draft'` on create, `team_id`, `event_id`, `track_id`
- `docs/invariants.md §I2, §I9`

## What We're Building

A team creates a submission for an event. Submissions start in `draft` status. Only one submission is allowed per team per event (I2). The event must be in `submissions_open` status (I9). Only team members can create a submission for their team. The caller's team is looked up automatically from the event — they don't pass teamId.

**Endpoint:** `POST /api/v1/events/{slug}/submissions`  
**Auth:** Required + must be a team member in this event

**Request:**
```json
{
  "title": "AI Hackathon Assistant",
  "description": "We built an AI agent that helps hackathon participants...",
  "trackId": "uuid",
  "repoUrl": "https://github.com/team/project",
  "demoUrl": "https://demo.project.com"
}
```

**Success (201):**
```json
{
  "data": {
    "id": "uuid",
    "teamId": "uuid",
    "eventId": "uuid",
    "trackId": "uuid",
    "title": "AI Hackathon Assistant",
    "description": "...",
    "repoUrl": "...",
    "demoUrl": "...",
    "coverUrl": null,
    "status": "draft",
    "finalScore": null,
    "overallRank": null,
    "trackRank": null,
    "submittedAt": null,
    "createdAt": "..."
  },
  "meta": {...}
}
```

**Error cases:**
| Condition | HTTP | Code |
|-----------|------|------|
| Caller has no team in this event | 403 | `FORBIDDEN` |
| Team already has a submission (I2) | 409 | `DUPLICATE_RESOURCE` |
| Event not in `submissions_open` status (I9) | 422 | `INVARIANT_VIOLATION` |
| trackId provided but doesn't belong to event | 400 | `VALIDATION_ERROR` |
| Title blank | 400 | `VALIDATION_ERROR` |

## Files to Create

### internal/submissions/domain/submission.go
```go
package domain

import (
    "errors"
    "fmt"
    "time"
    "github.com/google/uuid"
)

type SubmissionStatus string
const (
    StatusDraft        SubmissionStatus = "draft"
    StatusSubmitted    SubmissionStatus = "submitted"
    StatusDisqualified SubmissionStatus = "disqualified"
)

var ErrAlreadySubmitted  = errors.New("team already has a submission for this event")
var ErrSubmissionsNotOpen = errors.New("event is not accepting submissions")

type Submission struct {
    ID          uuid.UUID
    TeamID      uuid.UUID
    EventID     uuid.UUID
    TrackID     *uuid.UUID
    Title       string
    Description *string
    RepoURL     *string
    DemoURL     *string
    CoverURL    *string
    Status      SubmissionStatus
    FinalScore  *float64
    OverallRank *int
    TrackRank   *int
    SubmittedAt *time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

func NewDraftSubmission(teamID, eventID uuid.UUID, trackID *uuid.UUID, title string) (*Submission, error) {
    if title == "" {
        return nil, errors.New("title is required")
    }
    now := time.Now().UTC()
    return &Submission{
        ID:        uuid.New(),
        TeamID:    teamID,
        EventID:   eventID,
        TrackID:   trackID,
        Title:     title,
        Status:    StatusDraft,
        CreatedAt: now,
        UpdatedAt: now,
    }, nil
}

// CheckCanCreate validates that the event is accepting submissions (I9).
func CheckCanCreate(eventStatus string) error {
    if eventStatus != "submissions_open" {
        return fmt.Errorf("%w: event status is '%s'", ErrSubmissionsNotOpen, eventStatus)
    }
    return nil
}
```

### internal/submissions/port/in.go
```go
package port

import (
    "context"
    "github.com/google/uuid"
)

type CreateSubmissionUseCase interface {
    Create(ctx context.Context, cmd CreateSubmissionCommand) (*SubmissionDTO, error)
}

type CreateSubmissionCommand struct {
    CallerID    uuid.UUID
    EventSlug   string
    Title       string
    Description *string
    TrackID     *uuid.UUID
    RepoURL     *string
    DemoURL     *string
}
```

### internal/submissions/port/out.go
```go
package port

import (
    "context"
    "github.com/dogfood/platform/internal/submissions/domain"
    "github.com/google/uuid"
)

type SubmissionRepository interface {
    Save(ctx context.Context, sub *domain.Submission) error
    FindByTeamAndEvent(ctx context.Context, teamID, eventID uuid.UUID) (*domain.Submission, error)
    FindByID(ctx context.Context, id uuid.UUID) (*domain.Submission, error)
    Update(ctx context.Context, sub *domain.Submission) error
}

type EventReader interface {
    FindSummaryBySlug(ctx context.Context, slug string) (*EventSummary, error)
}

type TeamReader interface {
    FindByEventAndUser(ctx context.Context, eventID, userID uuid.UUID) (*TeamSummary, error)
}

type TrackReader interface {
    FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*TrackSummary, error)
}

type EventSummary struct {
    ID     uuid.UUID
    Status string
}
type TeamSummary struct {
    ID uuid.UUID
}
type TrackSummary struct {
    ID      uuid.UUID
    EventID uuid.UUID
}
```

### internal/submissions/usecase/create.go
```go
func (s *CreateSubmissionService) Create(ctx context.Context, cmd port.CreateSubmissionCommand) (*port.SubmissionDTO, error) {
    // 1. Load event
    event, err := s.events.FindSummaryBySlug(ctx, cmd.EventSlug)
    if err != nil { return nil, err }

    // 2. I9: event must be submissions_open
    if err := domain.CheckCanCreate(event.Status); err != nil {
        return nil, fmt.Errorf("%w: %s", response.ErrInvariantViolated, err.Error())
    }

    // 3. Find caller's team — 403 if they have no team
    team, err := s.teams.FindByEventAndUser(ctx, event.ID, cmd.CallerID)
    if err != nil || team == nil {
        return nil, fmt.Errorf("%w: must be a team member to submit", response.ErrForbidden)
    }

    // 4. I2: check team doesn't already have a submission
    existing, _ := s.subs.FindByTeamAndEvent(ctx, team.ID, event.ID)
    if existing != nil {
        return nil, fmt.Errorf("%w: team already has a submission", response.ErrInvariantViolated)
    }

    // 5. Validate trackID belongs to this event (if provided)
    if cmd.TrackID != nil {
        tracks, _ := s.tracks.FindByEventID(ctx, event.ID)
        found := false
        for _, t := range tracks {
            if t.ID == *cmd.TrackID { found = true; break }
        }
        if !found {
            return nil, fmt.Errorf("trackId does not belong to this event")
        }
    }

    // 6. Create draft submission
    sub, err := domain.NewDraftSubmission(team.ID, event.ID, cmd.TrackID, cmd.Title)
    if err != nil { return nil, err }
    sub.Description = cmd.Description
    sub.RepoURL = cmd.RepoURL
    sub.DemoURL = cmd.DemoURL

    if err := s.subs.Save(ctx, sub); err != nil { return nil, err }

    return toSubmissionDTO(sub), nil
}
```

## Tests Required

### Domain tests
- `TestNewDraftSubmission_ValidInput_ReturnsDraft`
- `TestNewDraftSubmission_EmptyTitle_ReturnsError`
- `TestCheckCanCreate_SubmissionsOpen_ReturnsNil`
- `TestCheckCanCreate_RegistrationOpen_ReturnsErrSubmissionsNotOpen` (I9)

### Use case tests
- `TestCreateSubmissionService_ValidInput_ReturnsDraftDTO`
- `TestCreateSubmissionService_NoTeam_Returns403`
- `TestCreateSubmissionService_EventNotOpen_Returns422` (I9)
- `TestCreateSubmissionService_DuplicateSubmission_Returns409` (I2)
- `TestCreateSubmissionService_InvalidTrackID_Returns400`

### Integration test
- Register, create team, transition event → create submission → 201 draft
- Create second submission for same team → 409 (I2)
- Try during `registration_open` → 422 (I9)

## Definition of Done
- [ ] `go test ./internal/submissions/...` → 100% green
- [ ] `POST /events/{slug}/submissions` → 201 with `status: "draft"`, `submittedAt: null`
- [ ] Duplicate submission per team → 409 (I2)
- [ ] Event not `submissions_open` → 422 (I9)
- [ ] No team → 403
- [ ] Invalid trackId → 400
