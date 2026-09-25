package port

import (
	"context"

	"github.com/dogfood-platform/dogfood/internal/submissions/domain"
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
	Slug   string
}

type TeamSummary struct {
	ID   uuid.UUID
	Name string
}

type TrackSummary struct {
	ID      uuid.UUID
	EventID uuid.UUID
	Name    string
}
