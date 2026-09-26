package domain_test

import (
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"

    "github.com/dogfood-platform/dogfood/internal/judging/domain"
)

func TestValidateScores_AllCriteriaCovered_ReturnsNil(t *testing.T) {
    c1 := uuid.New()
    c2 := uuid.New()

    criteria := []domain.RubricCriterion{
        {ID: c1, Name: "A", MaxScore: 10},
        {ID: c2, Name: "B", MaxScore: 5},
    }

    scores := []domain.ScoreInput{
        {CriterionID: c1, RawScore: 8},
        {CriterionID: c2, RawScore: 4},
    }

    err := domain.ValidateScores(scores, criteria)
    assert.NoError(t, err)
}

func TestValidateScores_MissingCriterion_ReturnsErrMissingCriteria(t *testing.T) {
    c1 := uuid.New()
    c2 := uuid.New()

    criteria := []domain.RubricCriterion{
        {ID: c1, MaxScore: 10},
        {ID: c2, MaxScore: 5},
    }

    scores := []domain.ScoreInput{
        {CriterionID: c1, RawScore: 8},
    }

    err := domain.ValidateScores(scores, criteria)
    assert.ErrorIs(t, err, domain.ErrMissingCriteria)
}

func TestValidateScores_ScoreExceedsMaxScore_ReturnsErrScoreOutOfRange(t *testing.T) {
    c1 := uuid.New()

    criteria := []domain.RubricCriterion{
        {ID: c1, MaxScore: 10},
    }

    scores := []domain.ScoreInput{
        {CriterionID: c1, RawScore: 11},
    }

    err := domain.ValidateScores(scores, criteria)
    assert.ErrorIs(t, err, domain.ErrScoreOutOfRange)
}

func TestValidateScores_UnknownCriterionID_ReturnsErrInvalidCriterionID(t *testing.T) {
    c1 := uuid.New()

    criteria := []domain.RubricCriterion{
        {ID: c1, MaxScore: 10},
    }

    scores := []domain.ScoreInput{
        {CriterionID: uuid.New(), RawScore: 8},
    }

    err := domain.ValidateScores(scores, criteria)
    assert.ErrorIs(t, err, domain.ErrInvalidCriterionID)
}

func TestWeightedTotal_ThreeCriteria_ReturnsCorrectSum(t *testing.T) {
    c1 := uuid.New()
    c2 := uuid.New()
    c3 := uuid.New()

    criteria := []domain.RubricCriterion{
        {ID: c1, MaxScore: 10, Weight: 0.5},
        {ID: c2, MaxScore: 5, Weight: 0.3},
        {ID: c3, MaxScore: 20, Weight: 0.2},
    }

    scores := []domain.ScoreInput{
        {CriterionID: c1, RawScore: 8},
        {CriterionID: c2, RawScore: 4},
        {CriterionID: c3, RawScore: 15},
    }

    total := domain.WeightedTotal(scores, criteria)
    assert.InDelta(t, 8.2, total, 0.001) // 8*0.5 + 4*0.3 + 15*0.2 = 4 + 1.2 + 3 = 8.2
}
