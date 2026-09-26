package port

import (
	"context"

	"github.com/google/uuid"
)

type CreateSubmissionUseCase interface {
	Create(ctx context.Context, cmd CreateSubmissionCommand) (*SubmissionDTO, error)
}

type CreateSubmissionCommand struct {
	CallerID    uuid.UUID
	EventSlug   string
	Title       string
	Description *string
	TrackID     *uuid.UUID
	RepoURL     *string
	DemoURL     *string
	VideoURL    *string
}

type SubmissionDTO struct {
	SubmissionID uuid.UUID `json:"submissionId"`
	Title        string    `json:"title"`
	Status       string    `json:"status"`
	TeamID       uuid.UUID `json:"teamId"`
	TeamName     string    `json:"teamName"`
	EventSlug    string    `json:"eventSlug"`
	Track        *TrackDTO `json:"track"`
	RepoURL      *string   `json:"repoUrl"`
	DemoURL      *string   `json:"demoUrl"`
	VideoURL     *string   `json:"videoUrl"`
	CoverURL     *string   `json:"coverImageUrl"`
	CreatedAt    string    `json:"createdAt"`
	UpdatedAt    string    `json:"updatedAt"`
	SubmittedAt  *string   `json:"submittedAt,omitempty"`
}

type TrackDTO struct {
	TrackID uuid.UUID `json:"trackId"`
	Name    string    `json:"name"`
}

type UpdateSubmissionUseCase interface {
	Update(ctx context.Context, cmd UpdateSubmissionCommand) (*SubmissionDTO, error)
}

type FinalSubmitUseCase interface {
	Submit(ctx context.Context, cmd SubmitCommand) (*SubmissionDTO, error)
}

type GetMySubmissionUseCase interface {
	GetMine(ctx context.Context, callerID uuid.UUID, eventSlug string) (*SubmissionDTO, error)
}

type UpdateSubmissionCommand struct {
	CallerID     uuid.UUID
	SubmissionID uuid.UUID
	Title        *string
	Description  *string
	RepoURL      *string
	DemoURL      *string
}

type SubmitCommand struct {
	CallerID     uuid.UUID
	SubmissionID uuid.UUID
}
