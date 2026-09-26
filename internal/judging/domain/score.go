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

func WeightedTotal(scores []ScoreInput, criteria []RubricCriterion) float64 {
    criteriaMap := make(map[uuid.UUID]RubricCriterion)
    for _, c := range criteria {
        criteriaMap[c.ID] = c
    }
    var total float64
    for _, s := range scores {
        c := criteriaMap[s.CriterionID]
        total += (float64(s.RawScore) / float64(c.MaxScore)) * c.Weight * float64(c.MaxScore)
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
