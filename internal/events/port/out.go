package port

import (
	"context"

	"github.com/dogfood-platform/dogfood/internal/events/domain"
	"github.com/google/uuid"
)

type EventRepository interface {
	Save(ctx context.Context, event *domain.Event) error
	FindBySlug(ctx context.Context, slug string) (*domain.Event, error)
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	ListPublished(ctx context.Context, page, pageSize int) ([]*domain.Event, int, error)
	GetEventDetail(ctx context.Context, slug string, callerID *uuid.UUID) (*domain.Event, []domain.Track, *string, error)
	Update(ctx context.Context, event *domain.Event) error
}

type EventRoleRepository interface {
	GrantRole(ctx context.Context, userID, eventID uuid.UUID, role string) error
	GetRole(ctx context.Context, userID, eventID uuid.UUID) (string, error)
}

type TrackRepository interface {
	FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*domain.Track, error)
	Save(ctx context.Context, track *domain.Track) error
}
