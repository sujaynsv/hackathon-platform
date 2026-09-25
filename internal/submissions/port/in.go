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
	SubmissionID  uuid.UUID  `json:"submissionId"`
	Title         string     `json:"title"`
	Status        string     `json:"status"`
	TeamID        uuid.UUID  `json:"teamId"`
	TeamName      string     `json:"teamName"`
	EventSlug     string     `json:"eventSlug"`
	Track         *TrackDTO  `json:"track"`
	RepoURL       *string    `json:"repoUrl"`
	DemoURL       *string    `json:"demoUrl"`
	VideoURL      *string    `json:"videoUrl"`
	CoverURL      *string    `json:"coverImageUrl"`
	CreatedAt     string     `json:"createdAt"`
	UpdatedAt     string     `json:"updatedAt"`
}

type TrackDTO struct {
	TrackID uuid.UUID `json:"trackId"`
	Name    string    `json:"name"`
}
