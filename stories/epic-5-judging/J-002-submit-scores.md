---
id: J-002
title: Submit Scores for Assignment
epic: judging
owner: Keerthika (backend)
status: "[ ] not-started"
branch: story/J-002-submit-scores
blocks: J-003, J-004
blocked-by: J-001, E-003
---

# J-002 · Submit Scores for Assignment

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules
- `MASTER-CONTEXT.md` — Invariant **I16**: only assigned judge can score; Invariant **I12**: scoring only allowed during `judging` window (before judging_deadline_at)
- `docs/api-design.md §Judging` — POST /judging/assignments/{id}/scores
- `docs/data-model.md §scores` — `UNIQUE(assignment_id, criterion_id)` — one score per criterion per assignment
- `docs/invariants.md §I12, §I16`

## What We're Building

A judge submits a score for every criterion in their rubric for a specific assignment. The scores must cover ALL rubric criteria (you can't partially score). When all criteria are scored, the assignment status automatically moves to `completed`. Scoring is only allowed before the judging deadline (I12) and only by the assigned judge (I16). Judges can update their scores before the deadline (re-submitting replaces scores).

**Endpoint:** `POST /api/v1/judging/assignments/{id}/scores`  
**Auth:** Must be the assigned judge for this assignment

**Request:**
```json
{
  "scores": [
    { "criterionId": "uuid", "rawScore": 8, "notes": "Good innovation, unique approach" },
    { "criterionId": "uuid", "rawScore": 7, "notes": "Strong execution" },
    { "criterionId": "uuid", "rawScore": 9, "notes": "Excellent presentation" }
  ]
}
```

**Validation rules:**
- Must provide a score for EVERY criterion in the rubric (count must match criteria count)
- `rawScore` must be between 1 and criterion's `maxScore` (inclusive)
- `criterionId` must belong to the rubric for this assignment

**Success (200):**
```json
{
  "data": {
    "assignmentId": "uuid",
    "status": "completed",
    "scores": [
      { "criterionId": "uuid", "criterionName": "Innovation", "rawScore": 8, "notes": "...", "weight": 0.30, "maxScore": 10 }
    ],
    "weightedTotal": 8.1
  },
  "meta": {...}
}
```

**Error cases:**
| Condition | HTTP | Code |
|-----------|------|------|
| Not the assigned judge (I16) | 404 | `NOT_FOUND` |
| Past judging deadline (I12) | 422 | `DEADLINE_PASSED` |
| Missing criteria in scores array | 400 | `VALIDATION_ERROR` |
| rawScore out of range | 400 | `VALIDATION_ERROR` |
| Unknown criterionId (not in rubric) | 400 | `VALIDATION_ERROR` |

## Files to Create

### internal/judging/domain/score.go
```go
package domain

import (
    "errors"
    "fmt"
    "time"
    "github.com/google/uuid"
)

var ErrScoringDeadlinePassed  = errors.New("judging deadline has passed")
var ErrScoreOutOfRange        = errors.New("score out of range")
var ErrMissingCriteria        = errors.New("must score all rubric criteria")
var ErrInvalidCriterionID     = errors.New("criterion does not belong to this rubric")

type Score struct {
    ID              uuid.UUID
    AssignmentID    uuid.UUID
    CriterionID     uuid.UUID
    JudgeID         uuid.UUID
    RawScore        int
    NormalizedScore *float64
    Notes           *string
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// ValidateScores checks all scores before persisting:
// 1. All criterion IDs are in the rubric
// 2. All criteria are covered (no missing ones)
// 3. Each score is within [1, maxScore]
func ValidateScores(scores []ScoreInput, criteria []RubricCriterion) error {
    criteriaMap := make(map[uuid.UUID]RubricCriterion)
    for _, c := range criteria {
        criteriaMap[c.ID] = c
    }

    scoredIDs := make(map[uuid.UUID]bool)
    for _, s := range scores {
        crit, ok := criteriaMap[s.CriterionID]
        if !ok {
            return fmt.Errorf("%w: %s", ErrInvalidCriterionID, s.CriterionID)
        }
        if s.RawScore < 1 || s.RawScore > crit.MaxScore {
            return fmt.Errorf("%w: got %d, expected 1–%d for criterion '%s'",
                ErrScoreOutOfRange, s.RawScore, crit.MaxScore, crit.Name)
        }
        scoredIDs[s.CriterionID] = true
    }

    if len(scoredIDs) < len(criteria) {
        return fmt.Errorf("%w: provided %d, required %d", ErrMissingCriteria, len(scoredIDs), len(criteria))
    }
    return nil
}

// WeightedTotal calculates the weighted score total.
// Used for display only — normalization (J-004) computes the final normalized scores.
func WeightedTotal(scores []ScoreInput, criteria []RubricCriterion) float64 {
    criteriaMap := make(map[uuid.UUID]RubricCriterion)
    for _, c := range criteria { criteriaMap[c.ID] = c }
    var total float64
    for _, s := range scores {
        c := criteriaMap[s.CriterionID]
        // Weighted score = (rawScore / maxScore) * weight * maxScore
        total += float64(s.RawScore) * c.Weight
    }
    return total
}

type ScoreInput struct {
    CriterionID uuid.UUID
    RawScore    int
    Notes       *string
}

type RubricCriterion struct {
    ID       uuid.UUID
    Name     string
    Weight   float64
    MaxScore int
}
```

### internal/judging/port/in.go (add)
```go
type SubmitScoresUseCase interface {
    SubmitScores(ctx context.Context, cmd SubmitScoresCommand) (*ScoreResultDTO, error)
}

type SubmitScoresCommand struct {
    JudgeID      uuid.UUID
    AssignmentID uuid.UUID
    Scores       []ScoreInput
}

type ScoreInput struct {
    CriterionID uuid.UUID
    RawScore    int
    Notes       *string
}
```

### internal/judging/usecase/scores.go
```go
func (s *SubmitScoresService) SubmitScores(ctx context.Context, cmd port.SubmitScoresCommand) (*port.ScoreResultDTO, error) {
    // 1. Load assignment — FindByIDAndJudge returns nil if wrong judge (I16)
    assignment, err := s.assignments.FindByIDAndJudge(ctx, cmd.AssignmentID, cmd.JudgeID)
    if err != nil || assignment == nil {
        return nil, fmt.Errorf("%w: assignment not found", response.ErrNotFound)
    }

    // 2. I12: check judging deadline
    event, _ := s.events.FindSummaryByID(ctx, assignment.EventID)
    if event.JudgingDeadlineAt != nil && time.Now().UTC().After(*event.JudgingDeadlineAt) {
        return nil, fmt.Errorf("%w: judging deadline has passed", response.ErrDeadlinePassed)
    }

    // 3. Load rubric criteria for this assignment
    criteria, _ := s.rubrics.FindCriteriaByRubricID(ctx, assignment.RubricID)

    // 4. Validate all scores (domain logic — pure function)
    domainScores := toDomainScoreInputs(cmd.Scores)
    domainCriteria := toDomainCriteria(criteria)
    if err := domain.ValidateScores(domainScores, domainCriteria); err != nil {
        return nil, err // 400 VALIDATION_ERROR
    }

    // 5. Upsert scores (INSERT ... ON CONFLICT UPDATE)
    if err := s.scores.UpsertAll(ctx, assignment.ID, cmd.JudgeID, cmd.Scores); err != nil {
        return nil, err
    }

    // 6. Mark assignment as completed
    if err := s.assignments.UpdateStatus(ctx, assignment.ID, domain.AssignmentCompleted); err != nil {
        return nil, err
    }

    // 7. Calculate weighted total for response (informational — not the normalized score)
    total := domain.WeightedTotal(domainScores, domainCriteria)

    return &port.ScoreResultDTO{
        AssignmentID:  assignment.ID.String(),
        Status:        "completed",
        Scores:        toScoreDTOs(cmd.Scores, criteria),
        WeightedTotal: total,
    }, nil
}
```

### internal/judging/repository/scores.go (UpsertAll)
```sql
-- UpsertAll: INSERT ... ON CONFLICT UPDATE (idempotent re-scoring)
INSERT INTO scores (id, assignment_id, criterion_id, judge_id, raw_score, notes, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
ON CONFLICT (assignment_id, criterion_id)
DO UPDATE SET raw_score = EXCLUDED.raw_score, notes = EXCLUDED.notes, updated_at = NOW();
```

## Tests Required

### Domain tests (domain/score_test.go)
- `TestValidateScores_AllCriteriaCovered_ReturnsNil`
- `TestValidateScores_MissingCriterion_ReturnsErrMissingCriteria`
- `TestValidateScores_ScoreExceedsMaxScore_ReturnsErrScoreOutOfRange`
- `TestValidateScores_UnknownCriterionID_ReturnsErrInvalidCriterionID`
- `TestWeightedTotal_ThreeCriteria_ReturnsCorrectSum`

### Use case tests
- `TestSubmitScoresService_ValidScores_Returns200_StatusCompleted`
- `TestSubmitScoresService_WrongJudge_Returns404` (I16)
- `TestSubmitScoresService_PastDeadline_Returns422` (I12)
- `TestSubmitScoresService_MissingCriterion_Returns400`
- `TestSubmitScoresService_ScoreOutOfRange_Returns400`
- `TestSubmitScoresService_Resubmit_UpdatesScores_StillCompleted` (idempotent)

### Integration test
- Assign judge → submit valid scores → verify scores in DB, status = 'completed'
- Submit again (update scores) → verify updated, still 'completed'
- Submit with wrong judge → 404

## Definition of Done
- [ ] `go test ./internal/judging/...` → 100% green
- [ ] `POST /judging/assignments/{id}/scores` with all criteria → 200, status = `completed`
- [ ] Missing criterion → 400 `VALIDATION_ERROR`
- [ ] Score > maxScore → 400 `VALIDATION_ERROR`
- [ ] Wrong judge → 404 (I16)
- [ ] Past deadline → 422 `DEADLINE_PASSED` (I12)
- [ ] Re-scoring updates scores (ON CONFLICT UPDATE) — idempotent
- [ ] `weightedTotal` in response matches expected calculation
