package port

import (
	"context"

	"github.com/google/uuid"
)

type RegisterForEventUseCase interface {
	Register(ctx context.Context, cmd RegisterCommand) (*RegistrationDTO, error)
}

type UnregisterFromEventUseCase interface {
	Unregister(ctx context.Context, cmd UnregisterCommand) error
}

type RegisterCommand struct {
	UserID    uuid.UUID
	EventSlug string
}

type UnregisterCommand struct {
	UserID    uuid.UUID
	EventSlug string
}

type RegistrationDTO struct {
	Registered   bool   `json:"registered"`
	EventID      string `json:"eventId"`
	EventSlug    string `json:"eventSlug"`
	RegisteredAt string `json:"registeredAt"`
}

type CreateTeamUseCase interface {
	Create(ctx context.Context, cmd CreateTeamCommand) (*TeamDTO, error)
}

type GetMyTeamUseCase interface {
	GetMyTeam(ctx context.Context, query GetMyTeamQuery) (*TeamDTO, error)
}

type CreateTeamCommand struct {
	UserID    uuid.UUID
	EventSlug string
	TeamName  string
}

type GetMyTeamQuery struct {
	UserID    uuid.UUID
	EventSlug string
}

type TeamDTO struct {
	ID         string          `json:"id"`
	EventID    string          `json:"eventId"`
	Name       string          `json:"name"`
	InviteCode string          `json:"inviteCode,omitempty"`
	IsLocked   bool            `json:"isLocked"`
	Members    []TeamMemberDTO `json:"members"`
	CreatedAt  string          `json:"createdAt"`
}

type TeamMemberDTO struct {
	ID       string `json:"id"`
	UserID   string `json:"userId"`
	Role     string `json:"role"`
	JoinedAt string `json:"joinedAt"`
}
