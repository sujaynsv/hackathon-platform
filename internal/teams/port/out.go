package port

import (
	"context"
	"time"

	"github.com/dogfood-platform/dogfood/internal/teams/domain"
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
	GetRole(ctx context.Context, userID, eventID uuid.UUID) (string, error)
	GrantParticipantRole(ctx context.Context, userID, eventID uuid.UUID) (time.Time, error)
	RevokeParticipantRole(ctx context.Context, userID, eventID uuid.UUID) error
}

type TeamMemberRepository interface {
	HasTeamInEvent(ctx context.Context, userID, eventID uuid.UUID) (bool, error)
}

type TeamRepository interface {
	FindByEventAndUser(ctx context.Context, eventID, userID uuid.UUID) (*domain.Team, error)
	ExistsByName(ctx context.Context, eventID uuid.UUID, name string) (bool, error)
	FindByInviteCode(ctx context.Context, code string) (*domain.Team, error)
	Save(ctx context.Context, team *domain.Team) error
	AddMember(ctx context.Context, member *domain.TeamMember) error
	GetMembers(ctx context.Context, teamID uuid.UUID) ([]domain.TeamMember, error)
}
