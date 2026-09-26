package port

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type CreateEventUseCase interface {
	Create(ctx context.Context, cmd CreateEventCommand) (*EventDTO, error)
}

type CreateEventCommand struct {
	OrganizerID          uuid.UUID
	Slug                 string
	Title                string
	Description          *string
	MaxTeamSize          int
	RegistrationOpensAt  *time.Time
	RegistrationClosesAt *time.Time
	SubmissionDeadlineAt *time.Time
	JudgingDeadlineAt    *time.Time
	VotingOpensAt        *time.Time
	VotingClosesAt       *time.Time
}

type ListEventsUseCase interface {
	List(ctx context.Context, q ListEventsQuery) (*EventListDTO, error)
}

type ListEventsQuery struct {
	CallerID *uuid.UUID // nil = anonymous
	Page     int
	PageSize int
}

type GetEventUseCase interface {
	GetBySlug(ctx context.Context, slug string, callerID *uuid.UUID) (*EventDetailDTO, error)
}

type EventDTO struct {
	ID                   uuid.UUID  `json:"id"`
	Slug                 string     `json:"slug"`
	Title                string     `json:"title"`
	Description          *string    `json:"description,omitempty"`
	BannerURL            *string    `json:"bannerUrl,omitempty"`
	OrganizerID          uuid.UUID  `json:"organizerId"`
	Status               string     `json:"status"`
	MaxTeamSize          int        `json:"maxTeamSize"`
	RegistrationOpensAt  *time.Time `json:"registrationOpensAt,omitempty"`
	RegistrationClosesAt *time.Time `json:"registrationClosesAt,omitempty"`
	SubmissionDeadlineAt *time.Time `json:"submissionDeadlineAt,omitempty"`
	JudgingDeadlineAt    *time.Time `json:"judgingDeadlineAt,omitempty"`
	VotingOpensAt        *time.Time `json:"votingOpensAt,omitempty"`
	VotingClosesAt       *time.Time `json:"votingClosesAt,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

type EventListDTO struct {
	Events     []EventDTO `json:"events"`
	Page       int        `json:"page"`
	PageSize   int        `json:"pageSize"`
	TotalCount int        `json:"totalCount"`
	TotalPages int        `json:"totalPages"`
}

type TrackDTO struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
}

type EventDetailDTO struct {
	EventDTO
	Tracks []TrackDTO `json:"tracks"`
	MyRole *string    `json:"myRole,omitempty"`
}

type UpdateEventUseCase interface {
	Update(ctx context.Context, cmd UpdateEventCommand) (*EventDetailDTO, error)
}

type UpdateEventCommand struct {
	CallerID             uuid.UUID
	Slug                 string
	Title                *string
	Description          *string
	NewStatus            *string
	MaxTeamSize          *int
	RegistrationOpensAt  *time.Time
	RegistrationClosesAt *time.Time
	SubmissionDeadlineAt *time.Time
	JudgingDeadlineAt    *time.Time
	VotingOpensAt        *time.Time
	VotingClosesAt       *time.Time
}
