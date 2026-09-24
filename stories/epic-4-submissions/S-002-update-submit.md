---
id: S-002
title: Update Submission + Final Submit — PATCH + POST /submit
epic: submissions
owner: Sujay (backend)
status: "[ ] not-started"
branch: story/S-002-update-submit
blocks: S-003, J-001
blocked-by: S-001
---

# S-002 · Update Submission + Final Submit

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules
- `MASTER-CONTEXT.md` — Invariant **I11**: cannot edit submission after deadline; Invariant **I13**: submission must be `draft` to be submitted (cannot re-submit once `submitted`)
- `docs/api-design.md §Submissions` — PATCH /submissions/{id}, POST /submissions/{id}/submit
- `docs/data-model.md §submissions` — `status`, `submitted_at`, `submission_deadline_at`
- `docs/invariants.md §I11, §I13`

## What We're Building

Teams edit their draft submission (title, description, URLs) and eventually click "Submit" to finalize it. After `POST /submit`, the status changes from `draft` → `submitted` and `submittedAt` is set. The submission deadline (I11) is checked before any edit or submission — past the deadline, nothing can change. A submitted submission cannot be re-submitted (I13).

**Endpoints:**
- `PATCH /api/v1/submissions/{id}` — update draft fields (team member only, before deadline)
- `POST /api/v1/submissions/{id}/submit` — finalize submission (team member, before deadline)
- `GET /api/v1/events/{slug}/submissions/mine` — get my team's submission

### PATCH /submissions/{id}
**Auth:** Must be team member who owns this submission  
**Request (all optional):**
```json
{ "title": "New Title", "description": "...", "repoUrl": "...", "demoUrl": "..." }
```
**Success (200):** Updated submission DTO  
**Errors:**
| Condition | HTTP | Code |
|-----------|------|------|
| Not team member | 403 | `FORBIDDEN` |
| Submission not found | 404 | `NOT_FOUND` |
| Past submission deadline (I11) | 422 | `DEADLINE_PASSED` |
| Status is not `draft` (I13) | 422 | `INVARIANT_VIOLATION` |

### POST /submissions/{id}/submit
**Auth:** Must be team member who owns this submission  
**Success (200):** `{ "data": { ...submissionDTO, "status": "submitted", "submittedAt": "..." }, "meta": {...} }`  
**Errors:** Same as PATCH, plus:
| Condition | HTTP | Code |
|-----------|------|------|
| Status already `submitted` (I13) | 409 | `DUPLICATE_RESOURCE` |

## Files to Create / Modify

### internal/submissions/domain/submission.go (add)
```go
var ErrDeadlinePassed    = errors.New("submission deadline has passed")
var ErrAlreadySubmitted  = errors.New("submission already finalized")
var ErrNotDraft          = errors.New("only draft submissions can be edited")

// CheckCanEdit validates that the submission can still be edited (I11 + I13).
func (s *Submission) CheckCanEdit(deadlineAt *time.Time) error {
    if s.Status != StatusDraft {
        return fmt.Errorf("%w", ErrNotDraft)
    }
    if deadlineAt != nil && time.Now().UTC().After(*deadlineAt) {
        return fmt.Errorf("%w: deadline was %s", ErrDeadlinePassed, deadlineAt.Format(time.RFC3339))
    }
    return nil
}

// Submit transitions a draft submission to submitted (I13).
func (s *Submission) Submit(deadlineAt *time.Time) error {
    if err := s.CheckCanEdit(deadlineAt); err != nil {
        return err
    }
    now := time.Now().UTC()
    s.Status = StatusSubmitted
    s.SubmittedAt = &now
    s.UpdatedAt = now
    return nil
}
```

### internal/submissions/port/in.go (add)
```go
type UpdateSubmissionUseCase interface {
    Update(ctx context.Context, cmd UpdateSubmissionCommand) (*SubmissionDTO, error)
}
type FinalSubmitUseCase interface {
    Submit(ctx context.Context, cmd SubmitCommand) (*SubmissionDTO, error)
}
type GetMySubmissionUseCase interface {
    GetMine(ctx context.Context, callerID uuid.UUID, eventSlug string) (*SubmissionDTO, error)
}

type UpdateSubmissionCommand struct {
    CallerID     uuid.UUID
    SubmissionID uuid.UUID
    Title        *string
    Description  *string
    RepoURL      *string
    DemoURL      *string
}
type SubmitCommand struct {
    CallerID     uuid.UUID
    SubmissionID uuid.UUID
}
```

### internal/submissions/usecase/update.go
```go
func (s *UpdateSubmissionService) Update(ctx context.Context, cmd port.UpdateSubmissionCommand) (*port.SubmissionDTO, error) {
    // 1. Load submission
    sub, err := s.subs.FindByID(ctx, cmd.SubmissionID)
    if err != nil { return nil, err }

    // 2. Verify caller is team member
    team, err := s.teams.FindByEventAndUser(ctx, sub.EventID, cmd.CallerID)
    if err != nil || team == nil || team.ID != sub.TeamID {
        return nil, fmt.Errorf("%w: not a member of this team", response.ErrForbidden)
    }

    // 3. Load event deadline
    event, _ := s.events.FindSummaryByID(ctx, sub.EventID)

    // 4. I11 + I13: check can edit
    if err := sub.CheckCanEdit(event.SubmissionDeadlineAt); err != nil {
        if errors.Is(err, domain.ErrDeadlinePassed) {
            return nil, fmt.Errorf("%w: %s", response.ErrDeadlinePassed, err.Error())
        }
        return nil, fmt.Errorf("%w: %s", response.ErrInvariantViolated, err.Error())
    }

    // 5. Apply updates
    if cmd.Title != nil { sub.Title = *cmd.Title }
    if cmd.Description != nil { sub.Description = cmd.Description }
    if cmd.RepoURL != nil { sub.RepoURL = cmd.RepoURL }
    if cmd.DemoURL != nil { sub.DemoURL = cmd.DemoURL }
    sub.UpdatedAt = time.Now().UTC()

    if err := s.subs.Update(ctx, sub); err != nil { return nil, err }
    return toSubmissionDTO(sub), nil
}
```

### internal/submissions/usecase/submit.go
```go
func (s *FinalSubmitService) Submit(ctx context.Context, cmd port.SubmitCommand) (*port.SubmissionDTO, error) {
    sub, err := s.subs.FindByID(ctx, cmd.SubmissionID)
    if err != nil { return nil, err }

    // Verify team membership
    team, _ := s.teams.FindByEventAndUser(ctx, sub.EventID, cmd.CallerID)
    if team == nil || team.ID != sub.TeamID {
        return nil, fmt.Errorf("%w: not a team member", response.ErrForbidden)
    }

    event, _ := s.events.FindSummaryByID(ctx, sub.EventID)

    // I11 + I13: use domain method — it handles both checks atomically
    if err := sub.Submit(event.SubmissionDeadlineAt); err != nil {
        if errors.Is(err, domain.ErrDeadlinePassed) {
            return nil, fmt.Errorf("%w", response.ErrDeadlinePassed)
        }
        if errors.Is(err, domain.ErrNotDraft) {
            return nil, fmt.Errorf("%w: submission already finalized", response.ErrInvariantViolated)
        }
        return nil, err
    }

    if err := s.subs.Update(ctx, sub); err != nil { return nil, err }
    return toSubmissionDTO(sub), nil
}
```

## Tests Required

### Domain tests
- `TestSubmission_CheckCanEdit_DraftBeforeDeadline_Nil`
- `TestSubmission_CheckCanEdit_AfterDeadline_ErrDeadlinePassed` (I11)
- `TestSubmission_CheckCanEdit_AlreadySubmitted_ErrNotDraft` (I13)
- `TestSubmission_Submit_DraftBeforeDeadline_SetsStatusAndTimestamp`
- `TestSubmission_Submit_AlreadySubmitted_ReturnsError` (I13)

### Use case tests
- `TestUpdateSubmission_ValidChanges_Returns200`
- `TestUpdateSubmission_NotTeamMember_Returns403`
- `TestUpdateSubmission_PastDeadline_Returns422_DEADLINE_PASSED` (I11)
- `TestUpdateSubmission_NotDraft_Returns422_INVARIANT_VIOLATION` (I13)
- `TestFinalSubmit_DraftBeforeDeadline_SetsSubmitted`
- `TestFinalSubmit_AlreadySubmitted_Returns409` (I13)
- `TestFinalSubmit_PastDeadline_Returns422` (I11)

### Integration test
- Create submission → update → submit → verify submittedAt set
- Try to update after submit → 422 (I13)
- Set deadline to past → try submit → 422 (I11)

## Definition of Done
- [ ] `go test ./internal/submissions/...` → 100% green
- [ ] `PATCH /submissions/{id}` → 200 updated DTO
- [ ] Past deadline → 422 `DEADLINE_PASSED` (I11)
- [ ] Not draft → 422 `INVARIANT_VIOLATION` (I13)
- [ ] `POST /submissions/{id}/submit` → 200 with `status:"submitted"`, `submittedAt` set
- [ ] Re-submit → 409 (I13)
- [ ] Non-team-member → 403 on both endpoints
