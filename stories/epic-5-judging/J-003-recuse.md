---
id: J-003
title: Recuse from Assignment
epic: judging
owner: Sujay (backend)
status: "[ ] not-started"
branch: story/J-003-recuse
blocks: J-004
blocked-by: J-002
---

# J-003 · Recuse from Assignment

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules
- `MASTER-CONTEXT.md` — Invariant **I16**: judge can only recuse their own assignment; Invariant **I17**: recuse action must be audit-logged
- `docs/api-design.md §Judging` — POST /judging/assignments/{id}/recuse
- `docs/data-model.md §judge_assignments` — `status = 'recused'`
- `docs/invariants.md §I16, §I17`

## What We're Building

A judge can recuse themselves from a specific assignment if they have a conflict of interest (e.g., they know the team). Recusal sets the assignment status to `recused`. A recused judge no longer needs to score that submission. The action is audit-logged (I17). A judge cannot recuse from a `completed` assignment (already scored). Only the assigned judge can recuse (I16).

**Endpoint:** `POST /api/v1/judging/assignments/{id}/recuse`  
**Auth:** Must be the assigned judge

**Request:** `{ "reason": "I know one of the team members personally" }`  
**Success (200):** `{ "data": { "assignmentId": "uuid", "status": "recused", "recusedAt": "..." }, "meta": {...} }`

**Error cases:**
| Condition | HTTP | Code |
|-----------|------|------|
| Not the assigned judge (I16) | 404 | `NOT_FOUND` |
| Assignment already `completed` | 422 | `INVARIANT_VIOLATION` |
| Assignment already `recused` | 409 | `DUPLICATE_RESOURCE` |
| Past judging deadline (I12) | 422 | `DEADLINE_PASSED` |

## Files to Create / Modify

### internal/judging/domain/assignment.go (add)
```go
var ErrAlreadyCompleted = errors.New("cannot recuse from a completed assignment")
var ErrAlreadyRecused   = errors.New("already recused from this assignment")

func (a *JudgeAssignment) Recuse() error {
    switch a.Status {
    case AssignmentCompleted:
        return ErrAlreadyCompleted
    case AssignmentRecused:
        return ErrAlreadyRecused
    }
    a.Status = AssignmentRecused
    return nil
}
```

### internal/judging/port/in.go (add)
```go
type RecuseUseCase interface {
    Recuse(ctx context.Context, cmd RecuseCommand) (*RecuseDTO, error)
}

type RecuseCommand struct {
    JudgeID      uuid.UUID
    AssignmentID uuid.UUID
    Reason       string
}

type RecuseDTO struct {
    AssignmentID string `json:"assignmentId"`
    Status       string `json:"status"`
    RecusedAt    string `json:"recusedAt"`
}
```

### internal/judging/usecase/recuse.go
```go
func (s *RecuseService) Recuse(ctx context.Context, cmd port.RecuseCommand) (*port.RecuseDTO, error) {
    // 1. Load assignment — 404 if wrong judge (I16)
    assignment, err := s.assignments.FindByIDAndJudge(ctx, cmd.AssignmentID, cmd.JudgeID)
    if err != nil || assignment == nil {
        return nil, fmt.Errorf("%w", response.ErrNotFound)
    }

    // 2. I12: check judging deadline (recuse not allowed after deadline)
    event, _ := s.events.FindSummaryByID(ctx, assignment.EventID)
    if event.JudgingDeadlineAt != nil && time.Now().UTC().After(*event.JudgingDeadlineAt) {
        return nil, fmt.Errorf("%w: judging deadline has passed", response.ErrDeadlinePassed)
    }

    // 3. Domain: validate recuse is allowed
    if err := assignment.Recuse(); err != nil {
        if errors.Is(err, domain.ErrAlreadyCompleted) {
            return nil, fmt.Errorf("%w: %s", response.ErrInvariantViolated, err.Error())
        }
        if errors.Is(err, domain.ErrAlreadyRecused) {
            return nil, fmt.Errorf("%w: already recused", response.ErrInvariantViolated)
        }
        return nil, err
    }

    // 4. Persist status change
    if err := s.assignments.UpdateStatus(ctx, assignment.ID, domain.AssignmentRecused); err != nil {
        return nil, err
    }

    // 5. I17: audit log
    now := time.Now().UTC()
    _ = s.auditLog.Write(ctx, &port.AuditEntry{
        ActorID:      cmd.JudgeID,
        Action:       "RECUSE_ASSIGNMENT",
        ResourceType: "judge_assignment",
        ResourceID:   assignment.ID,
        Changes:      map[string]any{"reason": cmd.Reason, "status": "recused"},
    })

    return &port.RecuseDTO{
        AssignmentID: assignment.ID.String(),
        Status:       "recused",
        RecusedAt:    now.Format(time.RFC3339),
    }, nil
}
```

## Tests Required

### Domain tests
- `TestAssignment_Recuse_PendingStatus_SetsRecused`
- `TestAssignment_Recuse_CompletedStatus_ReturnsErrAlreadyCompleted`
- `TestAssignment_Recuse_AlreadyRecused_ReturnsErrAlreadyRecused`

### Use case tests
- `TestRecuseService_ValidRecuse_Returns200`
- `TestRecuseService_WrongJudge_Returns404` (I16)
- `TestRecuseService_AlreadyCompleted_Returns422`
- `TestRecuseService_PastDeadline_Returns422` (I12)
- `TestRecuseService_WritesAuditLog` (I17)

### Integration test
- Submit scores → try to recuse completed assignment → 422
- Recuse pending assignment → 200, status = 'recused'
- Recuse again → 409

## Definition of Done
- [ ] `go test ./internal/judging/...` → 100% green
- [ ] `POST /judging/assignments/{id}/recuse` → 200, status = `recused`
- [ ] Wrong judge → 404 (I16)
- [ ] Completed assignment → 422 `INVARIANT_VIOLATION`
- [ ] Audit log entry written (I17) — mock verify in use case test
- [ ] Past deadline → 422 `DEADLINE_PASSED` (I12)
