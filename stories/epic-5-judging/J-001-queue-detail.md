---
id: J-001
title: Judge Queue + Submission Detail for Judges
epic: judging
owner: Sujay (backend)
status: "[ ] not-started"
branch: story/J-001-queue-detail
blocks: J-002
blocked-by: E-004, S-002
---

# J-001 · Judge Queue + Submission Detail for Judges

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules; use raw SQL JOINs for the queue query
- `MASTER-CONTEXT.md` — Invariant **I16**: only assigned judges can score a submission; judge assignments auto-created when event transitions to `judging`
- `docs/api-design.md §Judging` — GET /judging/queue, GET /judging/assignments/{id}
- `docs/data-model.md §judge_assignments, §submissions, §rubric_criteria` — assignment_status enum
- `docs/invariants.md §I16` — judge can only score their own assigned submissions

## What We're Building

When an event transitions to `judging` status, the system creates judge assignments — each submitted submission is assigned to every judge in the event. This story builds the assignment creation logic (triggered by the status transition) and the endpoints for judges to see their queue and view assignment details.

**Endpoints:**
- `GET /api/v1/judging/queue` — authenticated judge: list my assignments for events I'm judging
- `GET /api/v1/judging/assignments/{id}` — authenticated judge: detail of one assignment with submission + rubric

### Assignment auto-creation (triggered from events module)
When `Event.Transition(judging)` succeeds, the events use case calls a `JudgeAssignmentCreator` port:
```
For each submitted submission in the event:
  For each judge in event_roles WHERE role='judge':
    Create judge_assignments row (submissionId, judgeId, rubricId, status='pending')
```
This is done in a single SQL INSERT ... SELECT to avoid N+1.

### GET /judging/queue
**Auth:** Authenticated user (only sees their own assignments)  
**Query params:** `eventId` (optional), `status` (`pending` | `in_progress` | `completed` | `all`, default `pending`)  
**Response (200):**
```json
{
  "data": [
    {
      "id": "uuid",
      "submissionId": "uuid",
      "submissionTitle": "AI Hackathon Assistant",
      "teamName": "Team Rocket",
      "coverUrl": "...",
      "eventId": "uuid",
      "eventTitle": "Dogfood Hackathon 2026",
      "rubricId": "uuid",
      "status": "pending",
      "assignedAt": "..."
    }
  ],
  "meta": {...}
}
```

### GET /judging/assignments/{id}
**Auth:** Must be the assigned judge (I16 — other judges cannot view each other's assignments)  
**Response (200):**
```json
{
  "data": {
    "id": "uuid",
    "status": "in_progress",
    "submission": {
      "id": "uuid", "title": "...", "description": "...",
      "repoUrl": "...", "demoUrl": "...", "coverUrl": "...",
      "team": { "id": "uuid", "name": "Team Rocket" }
    },
    "rubric": {
      "id": "uuid", "name": "Main Rubric",
      "criteria": [
        { "id": "uuid", "name": "Innovation", "weight": 0.30, "maxScore": 10, "description": "..." }
      ]
    },
    "existingScores": [
      { "criterionId": "uuid", "rawScore": 8, "notes": "..." }
    ]
  },
  "meta": {...}
}
```
**Error:** Assignment not found OR caller is not the assigned judge → 404 (do not reveal existence to non-assigned judges)

## Files to Create

### internal/judging/domain/assignment.go
```go
package domain

import (
    "time"
    "github.com/google/uuid"
)

type AssignmentStatus string
const (
    AssignmentPending    AssignmentStatus = "pending"
    AssignmentInProgress AssignmentStatus = "in_progress"
    AssignmentCompleted  AssignmentStatus = "completed"
    AssignmentRecused    AssignmentStatus = "recused"
)

type JudgeAssignment struct {
    ID           uuid.UUID
    EventID      uuid.UUID
    JudgeID      uuid.UUID
    SubmissionID uuid.UUID
    RubricID     uuid.UUID
    Status       AssignmentStatus
    AssignedAt   time.Time
    CompletedAt  *time.Time
}
```

### internal/judging/port/in.go
```go
type GetQueueUseCase interface {
    GetQueue(ctx context.Context, judgeID uuid.UUID, q QueueQuery) (*AssignmentListDTO, error)
}
type GetAssignmentDetailUseCase interface {
    GetDetail(ctx context.Context, judgeID, assignmentID uuid.UUID) (*AssignmentDetailDTO, error)
}
type CreateAssignmentsUseCase interface {
    CreateForEvent(ctx context.Context, eventID uuid.UUID) error
}

type QueueQuery struct {
    EventID *uuid.UUID
    Status  string
    Page    int
    PageSize int
}
```

### internal/judging/port/out.go
```go
type AssignmentRepository interface {
    Save(ctx context.Context, a *domain.JudgeAssignment) error
    BulkCreateForEvent(ctx context.Context, eventID uuid.UUID) error // single INSERT...SELECT
    FindByIDAndJudge(ctx context.Context, id, judgeID uuid.UUID) (*domain.JudgeAssignment, error)
    ListByJudge(ctx context.Context, judgeID uuid.UUID, q port.QueueQuery) ([]*QueueRow, int, error)
    UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AssignmentStatus) error
}

type QueueRow struct {
    AssignmentID    uuid.UUID
    SubmissionID    uuid.UUID
    SubmissionTitle string
    TeamName        string
    CoverURL        *string
    EventID         uuid.UUID
    EventTitle      string
    RubricID        uuid.UUID
    Status          domain.AssignmentStatus
    AssignedAt      time.Time
}
```

### internal/judging/repository/postgres.go — BulkCreate SQL
```sql
-- BulkCreateForEvent: single INSERT...SELECT (no application-level loops)
INSERT INTO judge_assignments (id, event_id, judge_id, submission_id, rubric_id, status, assigned_at)
SELECT
    gen_random_uuid(),
    $1,                                -- event_id
    er.user_id,                        -- judge
    s.id,                              -- submission
    r.id,                              -- rubric
    'pending',
    NOW()
FROM submissions s
CROSS JOIN event_roles er
JOIN rubrics r ON r.event_id = $1
WHERE s.event_id = $1
  AND s.status = 'submitted'
  AND er.event_id = $1
  AND er.role = 'judge'
ON CONFLICT (judge_id, submission_id) DO NOTHING;
```

### internal/judging/usecase/queue.go
```go
func (s *GetQueueService) GetQueue(ctx context.Context, judgeID uuid.UUID, q port.QueueQuery) (*port.AssignmentListDTO, error) {
    rows, total, err := s.assignments.ListByJudge(ctx, judgeID, q)
    // Map rows to DTO, return with pagination meta
}

func (s *GetDetailService) GetDetail(ctx context.Context, judgeID, assignmentID uuid.UUID) (*port.AssignmentDetailDTO, error) {
    // FindByIDAndJudge — returns nil if judge doesn't own this assignment (I16)
    a, err := s.assignments.FindByIDAndJudge(ctx, assignmentID, judgeID)
    if err != nil || a == nil {
        return nil, fmt.Errorf("%w: assignment not found", response.ErrNotFound)
    }
    // Load submission detail, rubric with criteria, existing scores
    // Return full AssignmentDetailDTO
}
```

## Tests Required

### Use case tests
- `TestGetQueueService_OnlyReturnsCallerAssignments` (I16)
- `TestGetQueueService_FilterByStatus_PendingOnly`
- `TestGetAssignmentDetailService_WrongJudge_Returns404` (I16 — not 403, prevents disclosure)
- `TestGetAssignmentDetailService_CorrectJudge_ReturnsFullDetail`
- `TestCreateAssignmentsService_CreatesOnePerJudgePerSubmission`
- `TestCreateAssignmentsService_IdempotentOnDuplicateCall` — ON CONFLICT DO NOTHING

### Integration test
- Create event → add 2 judges → submit 3 submissions → transition to `judging`
- Each judge has 3 assignments (one per submission)
- GET /judging/queue as judge1 → 3 assignments, all `pending`
- GET /judging/assignments/{id} as judge2 (wrong judge) → 404

## Definition of Done
- [ ] `go test ./internal/judging/...` → 100% green
- [ ] Assignment creation uses single INSERT...SELECT (no Go loops over submissions × judges)
- [ ] `GET /judging/queue` → only caller's assignments
- [ ] `GET /judging/assignments/{id}` with wrong judge → 404 (not 403 — I16)
- [ ] Assignment detail includes submission, rubric criteria, and any existing scores
- [ ] `BulkCreateForEvent` is idempotent (safe to call twice — ON CONFLICT DO NOTHING)
