package port

import (
	"context"
	"time"

	sharedport "github.com/dogfood-platform/dogfood/internal/shared/port"
	"github.com/dogfood-platform/dogfood/internal/submissions/domain"
	"github.com/google/uuid"
)

type SubmissionRepository interface {
	Save(ctx context.Context, sub *domain.Submission) error
	FindByTeamAndEvent(ctx context.Context, teamID, eventID uuid.UUID) (*domain.Submission, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Submission, error)
	Update(ctx context.Context, sub *domain.Submission) error
	ListGallery(ctx context.Context, eventID uuid.UUID, trackID *uuid.UUID, page, pageSize int) ([]*SubmissionGalleryRow, int, error)
}

type SubmissionGalleryRow struct {
	SubmissionID uuid.UUID
	Title        string
	Status       string
	TeamID       uuid.UUID
	TeamName     string
	EventID      uuid.UUID
	TrackID      *uuid.UUID
	TrackName    *string
	RepoURL      *string
	DemoURL      *string
	VideoURL     *string
	CoverURL     *string
	FinalScore   *float64
	CreatedAt    time.Time
	UpdatedAt    time.Time
	SubmittedAt  *time.Time
}

type UploadRepository interface {
	Save(ctx context.Context, upload *domain.Upload) error
	FindBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]*domain.Upload, error)
	FindCoverBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*domain.Upload, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// FileStorage is re-exported from shared/port — the submissions module uses this interface
type FileStorage = sharedport.FileStorage


type EventReader interface {
	FindSummaryBySlug(ctx context.Context, slug string) (*EventSummary, error)
	FindSummaryByID(ctx context.Context, id uuid.UUID) (*EventSummary, error)
}

type TeamReader interface {
	FindByEventAndUser(ctx context.Context, eventID, userID uuid.UUID) (*TeamSummary, error)
}

type TrackReader interface {
	FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*TrackSummary, error)
}

type EventSummary struct {
	ID                   uuid.UUID
	Status               string
	Slug                 string
	SubmissionDeadlineAt *time.Time
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

type AuditLogWriter interface {
	Write(ctx context.Context, entry *AuditEntry) error
}

type AuditEntry struct {
	ActorID      uuid.UUID
	Action       string
	ResourceType string
	ResourceID   uuid.UUID
	Changes      map[string]any
}

type UserReader interface {
	IsAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
}
