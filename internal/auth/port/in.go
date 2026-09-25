package port

import "context"

// RegisterUseCase is the inbound port for user registration.
// The handler calls this interface — never the concrete RegisterService struct.
type RegisterUseCase interface {
	Register(ctx context.Context, cmd RegisterCommand) (*AuthResponse, error)
}

type VerifyEmailCommand struct {
	Token string `json:"token"`
}

type VerifyEmailUseCase interface {
	Verify(ctx context.Context, cmd VerifyEmailCommand) error
}

type RegisterCommand struct {
	Email       string
	Password    string
	DisplayName string
}

type AuthResponse struct {
	User                 UserDTO `json:"user"`
	AccessToken          string  `json:"accessToken,omitempty"`
	RefreshToken         string  `json:"refreshToken,omitempty"`
	RequiresVerification bool    `json:"requiresVerification,omitempty"`
}

type UserDTO struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl"`
	IsAdmin     bool    `json:"isAdmin"`
	CreatedAt   string  `json:"createdAt"`
}
