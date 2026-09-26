package port

import (
	"context"
	"time"
)

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

type LoginUseCase interface {
	Login(ctx context.Context, cmd LoginCommand) (*AuthResponse, error)
}

type RefreshUseCase interface {
	Refresh(ctx context.Context, cmd RefreshCommand) (*TokenPair, error)
}

type LogoutCommand struct {
	JTI             string
	ExpiresAt       time.Time
	RawRefreshToken string
}

type LogoutUseCase interface {
	Logout(ctx context.Context, cmd LogoutCommand) error
}

type RegisterCommand struct {
	Email        string
	Password     string
	DisplayName  string
	CaptchaToken string
}

type LoginCommand struct {
	Email    string
	Password string
}

type RefreshCommand struct {
	RawRefreshToken string
}

type TokenPair struct {
	AccessToken  string  `json:"accessToken"`
	RefreshToken string  `json:"refreshToken"`
	User         UserDTO `json:"user"`
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

type WebAuthnRegisterBeginCommand struct {
	UserID string `json:"userId"`
}

type WebAuthnRegisterFinishCommand struct {
	UserID    string `json:"userId"`
	SessionID string `json:"sessionId"`
	Body      []byte `json:"-"`
}

type WebAuthnLoginBeginCommand struct {
	Email string `json:"email"`
}

type WebAuthnLoginFinishCommand struct {
	Email     string `json:"email"`
	SessionID string `json:"sessionId"`
	Body      []byte `json:"-"`
}

type WebAuthnUseCase interface {
	RegisterBegin(ctx context.Context, cmd WebAuthnRegisterBeginCommand) (creationData any, sessionID string, err error)
	RegisterFinish(ctx context.Context, cmd WebAuthnRegisterFinishCommand) (*AuthResponse, error)
	LoginBegin(ctx context.Context, cmd WebAuthnLoginBeginCommand) (assertionData any, sessionID string, err error)
	LoginFinish(ctx context.Context, cmd WebAuthnLoginFinishCommand) (*AuthResponse, error)
}
