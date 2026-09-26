package port

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type EventSummary struct {
	ID                   uuid.UUID
	Slug                 string
	Status               string
	RegistrationClosesAt *time.Time
}

type EventReader interface {
	FindSummaryBySlug(ctx context.Context, slug string) (*EventSummary, error)
}

type ParticipantRepository interface {
	HasRole(ctx context.Context, userID, eventID uuid.UUID, role string) (bool, error)
	GrantParticipantRole(ctx context.Context, userID, eventID uuid.UUID) (time.Time, error)
	RevokeParticipantRole(ctx context.Context, userID, eventID uuid.UUID) error
}

type TeamMemberRepository interface {
	HasTeamInEvent(ctx context.Context, userID, eventID uuid.UUID) (bool, error)
}
