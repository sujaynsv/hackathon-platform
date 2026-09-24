---
id: E-003
title: Create Rubric + Add Criteria — Scoring Configuration
epic: events
owner: Sujay (backend)
status: "[ ] not-started"
branch: story/E-003-rubric
blocks: E-004, J-002
blocked-by: E-001
---

# E-003 · Create Rubric + Add Criteria — Scoring Configuration

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules
- `MASTER-CONTEXT.md` — Invariant **I14**: sum of all rubric criterion weights MUST equal 1.0 (enforced at use case layer + DB CHECK constraint)
- `docs/api-design.md §Judging` — request/response for POST /events/{slug}/rubric, POST /events/{slug}/rubric/criteria
- `docs/data-model.md §rubrics, §rubric_criteria` — `weight NUMERIC(5,4)`, `max_score INT`
- `docs/invariants.md §I14` — weight sum = 1.0 rule, enforced before saving

## What We're Building

Organizers configure how submissions will be scored by creating a rubric with weighted criteria. For example: "Innovation (30%), Execution (40%), Presentation (30%)" — these weights must always sum to exactly 1.0. The rubric is created first, then criteria are added. A validation endpoint can check the rubric is ready before judging begins.

**Endpoints:**
- `POST /api/v1/events/{slug}/rubric` — create rubric for an event (organizer-only)
- `POST /api/v1/events/{slug}/rubric/criteria` — add a criterion to the rubric (organizer-only)
- `GET /api/v1/events/{slug}/rubric` — get rubric with all criteria (any authenticated user)
- `POST /api/v1/events/{slug}/rubric/validate` — check that weights sum to 1.0 (organizer)

### POST /events/{slug}/rubric
**Request:** `{ "name": "Main Rubric" }`  
**Response (201):** `{ "data": { "id": "uuid", "eventId": "uuid", "name": "Main Rubric", "criteria": [] }, "meta": {...} }`  
**Rule:** One rubric per event. If one already exists → 409 `DUPLICATE_RESOURCE`

### POST /events/{slug}/rubric/criteria
**Request:** `{ "name": "Innovation", "description": "...", "weight": 0.30, "maxScore": 10 }`  
**Response (201):** Added criterion DTO  
**Rules:**
- `weight` must be > 0 and ≤ 1.0
- `maxScore` must be > 0
- Adding a criterion does NOT validate total weight (that happens at /validate or on judging start)

### GET /events/{slug}/rubric
**Response (200):**
```json
{
  "data": {
    "id": "uuid",
    "name": "Main Rubric",
    "isValid": true,  // true if sum of weights == 1.0
    "criteria": [
      { "id": "uuid", "name": "Innovation", "description": "...", "weight": 0.30, "maxScore": 10 },
      { "id": "uuid", "name": "Execution",  "weight": 0.40, "maxScore": 10 },
      { "id": "uuid", "name": "Presentation","weight": 0.30, "maxScore": 10 }
    ]
  },
  "meta": {...}
}
```

### POST /events/{slug}/rubric/validate (I14)
Validates that weights sum to exactly 1.0:  
**Success (200):** `{ "data": { "valid": true, "weightSum": 1.0 }, "meta": {...} }`  
**Failure (422):** `{ "error": { "code": "INVARIANT_VIOLATION", "message": "criterion weights must sum to 1.0, got 0.90" }, "meta": {...} }`

## Files to Create

### internal/judging/domain/rubric.go
```go
package domain

import (
    "errors"
    "fmt"
    "math"
    "time"
    "github.com/google/uuid"
)

var ErrWeightsNotSumToOne = errors.New("criterion weights must sum to 1.0")

type Rubric struct {
    ID        uuid.UUID
    EventID   uuid.UUID
    Name      string
    Criteria  []RubricCriterion
    CreatedAt time.Time
}

type RubricCriterion struct {
    ID          uuid.UUID
    RubricID    uuid.UUID
    Name        string
    Description *string
    Weight      float64  // stored as NUMERIC(5,4), e.g. 0.3000
    MaxScore    int
    CreatedAt   time.Time
}

// ValidateWeights checks that the sum of all criterion weights is exactly 1.0 (I14).
// Allows for floating point tolerance of ±0.001.
func (r *Rubric) ValidateWeights() error {
    if len(r.Criteria) == 0 {
        return fmt.Errorf("%w: rubric has no criteria", ErrWeightsNotSumToOne)
    }
    var sum float64
    for _, c := range r.Criteria {
        sum += c.Weight
    }
    if math.Abs(sum-1.0) > 0.001 {
        return fmt.Errorf("%w: got %.4f", ErrWeightsNotSumToOne, sum)
    }
    return nil
}
```

### internal/judging/port/in.go
```go
type CreateRubricUseCase interface {
    CreateRubric(ctx context.Context, cmd CreateRubricCommand) (*RubricDTO, error)
}
type AddCriterionUseCase interface {
    AddCriterion(ctx context.Context, cmd AddCriterionCommand) (*CriterionDTO, error)
}
type GetRubricUseCase interface {
    GetRubric(ctx context.Context, eventSlug string) (*RubricDTO, error)
}
type ValidateRubricUseCase interface {
    Validate(ctx context.Context, eventSlug string) (*ValidationDTO, error)
}

type CreateRubricCommand struct {
    CallerID  uuid.UUID
    EventSlug string
    Name      string
}
type AddCriterionCommand struct {
    CallerID    uuid.UUID
    EventSlug   string
    Name        string
    Description *string
    Weight      float64
    MaxScore    int
}
```

### internal/judging/usecase/rubric.go
```go
func (s *CreateRubricService) CreateRubric(ctx context.Context, cmd port.CreateRubricCommand) (*port.RubricDTO, error) {
    // 1. Verify caller is organizer for this event
    // 2. Check rubric doesn't already exist for event → 409 if it does
    // 3. Create Rubric{} domain struct
    // 4. Persist
    // 5. Return DTO
}

func (s *AddCriterionService) AddCriterion(ctx context.Context, cmd port.AddCriterionCommand) (*port.CriterionDTO, error) {
    // 1. Verify organizer role
    // 2. Validate weight > 0 and ≤ 1.0
    // 3. Validate maxScore > 0
    // 4. Load rubric
    // 5. Save criterion
    // 6. Return DTO
}

func (s *ValidateRubricService) Validate(ctx context.Context, eventSlug string) (*port.ValidationDTO, error) {
    // 1. Load rubric with all criteria
    // 2. Call rubric.ValidateWeights() (I14 domain logic)
    // 3. Return {valid: bool, weightSum: float64}
}
```

## Tests Required

### Domain tests (domain/rubric_test.go)
- `TestRubric_ValidateWeights_SumsToOne_ReturnsNil`
- `TestRubric_ValidateWeights_SumsToPointNine_ReturnsErrWeightsNotSumToOne` (I14)
- `TestRubric_ValidateWeights_EmptyCriteria_ReturnsError`
- `TestRubric_ValidateWeights_FloatTolerance_PointNineNineNine_PassesAsOne` (0.999 within ±0.001)

### Use case tests
- `TestCreateRubricService_NonOrganizer_Returns403`
- `TestCreateRubricService_DuplicateRubric_Returns409`
- `TestAddCriterionService_InvalidWeight_Zero_Returns400`
- `TestAddCriterionService_InvalidWeight_GreaterThanOne_Returns400`
- `TestValidateRubricService_WeightsNotSumToOne_Returns422` (I14)
- `TestValidateRubricService_WeightsSumToOne_Returns200_Valid`

### Integration test
- Create event → create rubric → add 3 criteria (0.30, 0.40, 0.30) → validate → 200 valid
- Add criteria (0.30, 0.40) → validate → 422 (weights = 0.70, not 1.0)

## Definition of Done
- [ ] `go test ./internal/judging/...` → 100% green
- [ ] `POST /events/{slug}/rubric` → 201 rubric DTO
- [ ] Duplicate rubric → 409 `DUPLICATE_RESOURCE`
- [ ] `POST /events/{slug}/rubric/criteria` → 201 criterion DTO
- [ ] Weight = 0 or > 1.0 → 400 `VALIDATION_ERROR`
- [ ] `POST /events/{slug}/rubric/validate` with sum ≠ 1.0 → 422 `INVARIANT_VIOLATION` (I14)
- [ ] `GET /events/{slug}/rubric` returns criteria array with `isValid` flag
