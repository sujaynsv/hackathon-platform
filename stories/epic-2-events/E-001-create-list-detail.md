---
id: E-001
title: Create Event + List Events + Get Event Detail
epic: events
owner: Keerthika (backend)
status: "[ ] not-started"
branch: story/E-001-events-crud
blocks: E-002, E-003, T-001
blocked-by: A-002
---

# E-001 · Create Event + List Events + Get Event Detail

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules: domain/, port/, usecase/, handler/, repository/
- `MASTER-CONTEXT.md` — Invariant **I15**: event state machine (only valid transitions allowed); confirmed Go stack
- `docs/api-design.md §Events` — exact request/response shapes for POST /events, GET /events, GET /events/{slug}
- `docs/data-model.md §events, §tracks, §event_roles` — table schemas, all column names
- `docs/invariants.md §I15` — EventStatus transition table; organizer role automatically granted on create

## What We're Building

Three endpoints that form the core of event management. An organizer creates an event (starts as `draft`, organizer gets the `organizer` role in `event_roles` automatically). Any user can list published events (status != `draft`). Any user can get event detail by slug, with their own role included if authenticated.

**Endpoints:**
- `POST /api/v1/events` — authenticated, requires no special role (any logged-in user can create)
- `GET /api/v1/events` — public, returns all non-draft events paginated
- `GET /api/v1/events/{slug}` — public, returns event + tracks + caller's role if authenticated

### POST /api/v1/events
**Request:**
```json
{
  "slug": "hackathon-2026",
  "title": "Dogfood Hackathon 2026",
  "description": "Build something amazing",
  "maxTeamSize": 5,
  "registrationOpensAt": "2026-10-01T00:00:00Z",
  "registrationClosesAt": "2026-10-15T00:00:00Z",
  "submissionDeadlineAt": "2026-11-01T00:00:00Z",
  "judgingDeadlineAt": "2026-11-15T00:00:00Z",
  "votingOpensAt": "2026-11-16T00:00:00Z",
  "votingClosesAt": "2026-11-20T00:00:00Z"
}
```
**Success (201):** `{ "data": { "id": "uuid", "slug": "...", "title": "...", "status": "draft", "organizerId": "...", ... }, "meta": {...} }`

**Errors:**
| Condition | HTTP | Code |
|-----------|------|------|
| Slug already taken | 409 | `DUPLICATE_RESOURCE` |
| Missing title or slug | 400 | `VALIDATION_ERROR` |

### GET /api/v1/events
**Query params:** `page` (default 1), `pageSize` (default 20, max 50)  
**Response (200):** `{ "data": [Event...], "meta": { "page":1, "pageSize":20, "totalCount":42, "totalPages":3, ... } }`  
**Note:** Public users see only events with status != `draft`. Authenticated users also see `draft` events they organize.

### GET /api/v1/events/{slug}
**Response (200):**
```json
{
  "data": {
    "id": "uuid", "slug": "hackathon-2026", "title": "...", "status": "registration_open",
    "tracks": [{ "id": "uuid", "name": "Track A", "description": "..." }],
    "myRole": "participant",  // null if not authenticated or not registered
    "organizerId": "uuid",
    ...
  },
  "meta": {...}
}
```
**Error:** slug not found → 404 `NOT_FOUND`

## Files to Create

### internal/events/domain/event.go
```go
package domain

import (
    "errors"
    "fmt"
    "time"
    "github.com/google/uuid"
)

type EventStatus string
const (
    StatusDraft             EventStatus = "draft"
    StatusRegistrationOpen  EventStatus = "registration_open"
    StatusSubmissionsOpen   EventStatus = "submissions_open"
    StatusJudging           EventStatus = "judging"
    StatusVoting            EventStatus = "voting"
    StatusResultsPublished  EventStatus = "results_published"
    StatusArchived          EventStatus = "archived"
)

// allowedTransitions is the single source of truth for I15.
var allowedTransitions = map[EventStatus][]EventStatus{
    StatusDraft:            {StatusRegistrationOpen},
    StatusRegistrationOpen: {StatusSubmissionsOpen},
    StatusSubmissionsOpen:  {StatusJudging},
    StatusJudging:          {StatusVoting, StatusResultsPublished},
    StatusVoting:           {StatusResultsPublished},
    StatusResultsPublished: {StatusArchived},
    StatusArchived:         {},
}

var ErrInvalidTransition = errors.New("invalid state transition")
var ErrSlugTaken        = errors.New("event slug already taken")

type Event struct {
    ID                   uuid.UUID
    Slug                 string
    Title                string
    Description          *string
    BannerURL            *string
    OrganizerID          uuid.UUID
    Status               EventStatus
    NormalizationStatus  string
    RegistrationOpensAt  *time.Time
    RegistrationClosesAt *time.Time
    SubmissionDeadlineAt *time.Time
    JudgingDeadlineAt    *time.Time
    VotingOpensAt        *time.Time
    VotingClosesAt       *time.Time
    MaxTeamSize          int
    CreatedAt            time.Time
    UpdatedAt            time.Time
}

// Transition validates and applies a state transition (I15).
func (e *Event) Transition(target EventStatus) error {
    allowed := allowedTransitions[e.Status]
    for _, a := range allowed {
        if a == target {
            e.Status = target
            e.UpdatedAt = time.Now().UTC()
            return nil
        }
    }
    return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, e.Status, target)
}

func NewEvent(slug, title string, organizerID uuid.UUID, maxTeamSize int) (*Event, error) {
    if slug == "" { return nil, errors.New("slug is required") }
    if title == "" { return nil, errors.New("title is required") }
    if maxTeamSize < 1 { maxTeamSize = 5 }
    now := time.Now().UTC()
    return &Event{
        ID:                  uuid.New(),
        Slug:                slug,
        Title:               title,
        OrganizerID:         organizerID,
        Status:              StatusDraft,
        NormalizationStatus: "awaiting_judging",
        MaxTeamSize:         maxTeamSize,
        CreatedAt:           now,
        UpdatedAt:           now,
    }, nil
}
```

### internal/events/domain/track.go
```go
package domain

import (
    "time"
    "github.com/google/uuid"
)

type Track struct {
    ID          uuid.UUID
    EventID     uuid.UUID
    Name        string
    Description *string
    CreatedAt   time.Time
}
```

### internal/events/port/in.go
```go
package port

import (
    "context"
    "time"
    "github.com/google/uuid"
)

type CreateEventUseCase interface {
    Create(ctx context.Context, cmd CreateEventCommand) (*EventDTO, error)
}

type CreateEventCommand struct {
    OrganizerID          uuid.UUID
    Slug                 string
    Title                string
    Description          *string
    MaxTeamSize          int
    RegistrationOpensAt  *time.Time
    RegistrationClosesAt *time.Time
    SubmissionDeadlineAt *time.Time
    JudgingDeadlineAt    *time.Time
    VotingOpensAt        *time.Time
    VotingClosesAt       *time.Time
}

type ListEventsUseCase interface {
    List(ctx context.Context, q ListEventsQuery) (*EventListDTO, error)
}

type ListEventsQuery struct {
    CallerID *uuid.UUID // nil = anonymous
    Page     int
    PageSize int
}

type GetEventUseCase interface {
    GetBySlug(ctx context.Context, slug string, callerID *uuid.UUID) (*EventDetailDTO, error)
}
```

### internal/events/port/out.go
```go
package port

import (
    "context"
    "github.com/dogfood/platform/internal/events/domain"
    "github.com/google/uuid"
)

type EventRepository interface {
    Save(ctx context.Context, event *domain.Event) error
    FindBySlug(ctx context.Context, slug string) (*domain.Event, error)
    ExistsBySlug(ctx context.Context, slug string) (bool, error)
    ListPublished(ctx context.Context, page, pageSize int) ([]*domain.Event, int, error)
}

type EventRoleRepository interface {
    GrantRole(ctx context.Context, userID, eventID uuid.UUID, role string) error
    GetRole(ctx context.Context, userID, eventID uuid.UUID) (string, error)
}

type TrackRepository interface {
    FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*domain.Track, error)
    Save(ctx context.Context, track *domain.Track) error
}
```

### internal/events/usecase/create.go
```go
package usecase

import (
    "context"
    "fmt"
    "github.com/dogfood/platform/internal/events/domain"
    "github.com/dogfood/platform/internal/events/port"
    "github.com/dogfood/platform/internal/shared/response"
)

type CreateEventService struct {
    events port.EventRepository
    roles  port.EventRoleRepository
}

func (s *CreateEventService) Create(ctx context.Context, cmd port.CreateEventCommand) (*port.EventDTO, error) {
    // 1. Check slug uniqueness
    exists, err := s.events.ExistsBySlug(ctx, cmd.Slug)
    if err != nil {
        return nil, err
    }
    if exists {
        return nil, fmt.Errorf("%w: slug '%s' is already taken", response.ErrInvariantViolated, cmd.Slug)
    }

    // 2. Create domain event
    event, err := domain.NewEvent(cmd.Slug, cmd.Title, cmd.OrganizerID, cmd.MaxTeamSize)
    if err != nil {
        return nil, err
    }
    // Set optional date fields
    event.Description = cmd.Description
    event.RegistrationOpensAt = cmd.RegistrationOpensAt
    event.RegistrationClosesAt = cmd.RegistrationClosesAt
    event.SubmissionDeadlineAt = cmd.SubmissionDeadlineAt
    event.JudgingDeadlineAt = cmd.JudgingDeadlineAt
    event.VotingOpensAt = cmd.VotingOpensAt
    event.VotingClosesAt = cmd.VotingClosesAt

    // 3. Persist
    if err := s.events.Save(ctx, event); err != nil {
        return nil, fmt.Errorf("save event: %w", err)
    }

    // 4. Grant organizer role to creator
    if err := s.roles.GrantRole(ctx, cmd.OrganizerID, event.ID, "organizer"); err != nil {
        return nil, fmt.Errorf("grant organizer role: %w", err)
    }

    return toEventDTO(event), nil
}
```

### internal/events/repository/postgres.go
Key implementation notes:
- Use **sqlx** for simple CRUD (Save, FindBySlug, ExistsBySlug)
- Use **raw SQL with JOINs** for the list and detail queries (no N+1)
- The `GET /events/{slug}` detail query must JOIN `events`, `tracks`, and `event_roles` in ONE query:
```sql
SELECT
    e.*,
    t.id AS track_id, t.name AS track_name, t.description AS track_desc,
    er.role AS my_role
FROM events e
LEFT JOIN tracks t ON t.event_id = e.id
LEFT JOIN event_roles er ON er.event_id = e.id AND er.user_id = $2
WHERE e.slug = $1;
```
Scan results: group by event, collect tracks slice, extract `my_role` from first row.

- The `GET /events` list query must use OFFSET pagination with total count:
```sql
SELECT *, COUNT(*) OVER() AS total_count
FROM events
WHERE status != 'draft'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
```

## Tests Required

### Domain tests (domain/event_test.go)
- `TestEvent_Transition_ValidPath_Succeeds` — draft → registration_open
- `TestEvent_Transition_InvalidSkip_ReturnsErrInvalidTransition` — draft → judging
- `TestEvent_Transition_Backward_ReturnsErrInvalidTransition` — registration_open → draft
- `TestNewEvent_EmptySlug_ReturnsError`

### Use case tests (usecase/create_test.go)
- `TestCreateEventService_ValidInput_Returns201DTO`
- `TestCreateEventService_DuplicateSlug_ReturnsErrInvariantViolated`
- `TestCreateEventService_GrantsOrganizerRoleAfterCreate`

### Handler tests
- `TestCreateEventHandler_ValidBody_Returns201`
- `TestCreateEventHandler_MissingTitle_Returns400`
- `TestCreateEventHandler_UnauthenticatedUser_Returns401`
- `TestListEventsHandler_PublicUser_ExcludesDraftEvents`
- `TestGetEventHandler_UnknownSlug_Returns404`
- `TestGetEventHandler_AuthenticatedUser_IncludesMyRole`

### Integration test
- Full stack: create event → list → get by slug → verify myRole = "organizer" for creator

## Definition of Done
- [ ] `go test ./internal/events/...` → 100% green
- [ ] `POST /api/v1/events` → 201 with event in `draft` status
- [ ] Duplicate slug → 409 `DUPLICATE_RESOURCE`
- [ ] `GET /api/v1/events` — public users never see `draft` events
- [ ] `GET /api/v1/events/{slug}` — tracks array present, myRole populated when authenticated
- [ ] List query uses single SQL (no N+1) — verified by query count in integration test
- [ ] Detail query uses single JOIN SQL — verified by query count
- [ ] I15 `Transition()` tested: invalid transitions return `ErrInvalidTransition`
