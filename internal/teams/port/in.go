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
