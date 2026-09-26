package usecase

import (
    "context"
    "fmt"
    "time"

    "github.com/dogfood-platform/dogfood/internal/judging/domain"
    "github.com/dogfood-platform/dogfood/internal/judging/port"
    "github.com/dogfood-platform/dogfood/internal/shared/response"
)

type SubmitScoresService struct {
    assignments port.AssignmentRepository
    events      port.EventReader
    rubrics     port.RubricReader
    scores      port.ScoreWriter
}

func NewSubmitScoresService(
    assignments port.AssignmentRepository,
    events port.EventReader,
    rubrics port.RubricReader,
    scores port.ScoreWriter,
) *SubmitScoresService {
    return &SubmitScoresService{
        assignments: assignments,
        events:      events,
        rubrics:     rubrics,
        scores:      scores,
    }
}

func (s *SubmitScoresService) SubmitScores(ctx context.Context, cmd port.SubmitScoresCommand) (*port.ScoreResultDTO, error) {
    // 1. Load assignment — FindByIDAndJudge returns nil if wrong judge (I16)
    assignment, err := s.assignments.FindByIDAndJudge(ctx, cmd.AssignmentID, cmd.JudgeID)
    if err != nil {
        return nil, fmt.Errorf("%w: assignment not found", response.ErrNotFound)
    }
    if assignment == nil {
        return nil, fmt.Errorf("%w: assignment not found", response.ErrNotFound)
    }

    // 2. I12: check judging deadline
    event, err := s.events.FindSummaryByID(ctx, assignment.EventID)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch event summary: %w", err)
    }
    if event.JudgingDeadlineAt != nil && time.Now().UTC().After(*event.JudgingDeadlineAt) {
        return nil, fmt.Errorf("%w: judging deadline has passed", response.ErrDeadlinePassed)
    }

    // 3. Load rubric criteria for this assignment
    criteria, err := s.rubrics.FindCriteriaByRubricID(ctx, assignment.RubricID)
    if err != nil {
        return nil, fmt.Errorf("failed to load criteria: %w", err)
    }

    // 4. Validate all scores (domain logic — pure function)
    domainScores := toDomainScoreInputs(cmd.Scores)
    domainCriteria := toDomainCriteria(criteria)
    if err := domain.ValidateScores(domainScores, domainCriteria); err != nil {
        return nil, fmt.Errorf("%w: %v", response.ErrInvariantViolated, err)
    }

    // 5. Upsert scores (INSERT ... ON CONFLICT UPDATE)
    if err := s.scores.UpsertAll(ctx, assignment.ID, cmd.JudgeID, cmd.Scores); err != nil {
        return nil, fmt.Errorf("failed to upsert scores: %w", err)
    }

    // 6. Mark assignment as completed
    if err := s.assignments.UpdateStatus(ctx, assignment.ID, domain.AssignmentCompleted); err != nil {
        return nil, fmt.Errorf("failed to update assignment status: %w", err)
    }

    // 7. Calculate weighted total for response
    total := domain.WeightedTotal(domainScores, domainCriteria)

    return &port.ScoreResultDTO{
        AssignmentID:  assignment.ID.String(),
        Status:        "completed",
        Scores:        toScoreDTOs(cmd.Scores, criteria),
        WeightedTotal: total,
    }, nil
}

func toDomainScoreInputs(in []port.ScoreInput) []domain.ScoreInput {
    out := make([]domain.ScoreInput, len(in))
    for i, s := range in {
        out[i] = domain.ScoreInput{
            CriterionID: s.CriterionID,
            RawScore:    s.RawScore,
            Notes:       s.Notes,
        }
    }
    return out
}

func toDomainCriteria(in []port.CriterionDTO) []domain.RubricCriterion {
    out := make([]domain.RubricCriterion, len(in))
    for i, c := range in {
        out[i] = domain.RubricCriterion{
            ID:       c.ID,
            Name:     c.Name,
            Weight:   c.Weight,
            MaxScore: c.MaxScore,
        }
    }
    return out
}

func toScoreDTOs(inputs []port.ScoreInput, criteria []port.CriterionDTO) []port.ScoreDTO {
    out := make([]port.ScoreDTO, len(inputs))
    for i, in := range inputs {
        notes := ""
        if in.Notes != nil {
            notes = *in.Notes
        }
        out[i] = port.ScoreDTO{
            CriterionID: in.CriterionID,
            RawScore:    in.RawScore,
            Notes:       notes,
        }
    }
    return out
}
