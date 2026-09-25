package usecase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

type WebAuthnService struct {
	users         port.UserRepository
	creds         port.WebAuthnRepository
	cache         port.Cache
	tokens        port.TokenIssuer
	refreshTokens port.RefreshTokenRepository
	w             *webauthn.WebAuthn
}

func NewWebAuthnService(
	users port.UserRepository, 
	creds port.WebAuthnRepository, 
	cache port.Cache, 
	tokens port.TokenIssuer,
	refreshTokens port.RefreshTokenRepository,
	rpDisplayName, rpID, rpOrigin string) (*WebAuthnService, error) {
	w, err := webauthn.New(&webauthn.Config{
		RPDisplayName: rpDisplayName,
		RPID:          rpID,
		RPOrigins:     []string{rpOrigin},
	})
	if err != nil {
		return nil, fmt.Errorf("webauthn config: %w", err)
	}

	return &WebAuthnService{
		users:         users,
		creds:         creds,
		cache:         cache,
		tokens:        tokens,
		refreshTokens: refreshTokens,
		w:             w,
	}, nil
}

// webAuthnUserAdapter bridges domain.User and domain.WebAuthnCredential to webauthn.User interface
type webAuthnUserAdapter struct {
	u     *domain.User
	creds []*domain.WebAuthnCredential
}

func (w *webAuthnUserAdapter) WebAuthnID() []byte {
	id, _ := w.u.ID.MarshalBinary()
	return id
}

func (w *webAuthnUserAdapter) WebAuthnName() string {
	return w.u.Email
}

func (w *webAuthnUserAdapter) WebAuthnDisplayName() string {
	return w.u.DisplayName
}

func (w *webAuthnUserAdapter) WebAuthnIcon() string {
	if w.u.AvatarURL != nil {
		return *w.u.AvatarURL
	}
	return ""
}

func (w *webAuthnUserAdapter) WebAuthnCredentials() []webauthn.Credential {
	res := make([]webauthn.Credential, len(w.creds))
	for i, c := range w.creds {
		transports := make([]protocol.AuthenticatorTransport, len(c.Transport))
		for j, t := range c.Transport {
			transports[j] = protocol.AuthenticatorTransport(t)
		}
		res[i] = webauthn.Credential{
			ID:              c.CredentialID,
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Transport:       transports,
			Flags: webauthn.CredentialFlags{
				UserPresent:    c.Flags.UserPresent,
				UserVerified:   c.Flags.UserVerified,
				BackupEligible: c.Flags.BackupEligible,
				BackupState:    c.Flags.BackupState,
			},
			Authenticator: webauthn.Authenticator{
				AAGUID:       c.AuthenticatorAAGUID,
				SignCount:    c.SignCount,
				CloneWarning: c.CloneWarning,
			},
		}
	}
	return res
}

func (s *WebAuthnService) getUserAdapter(ctx context.Context, email string) (*webAuthnUserAdapter, error) {
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, err // Returns ErrNotFound mapped correctly
	}

	if !u.IsVerified {
		return nil, fmt.Errorf("%w: user email not verified", response.ErrForbidden)
	}

	creds, err := s.creds.FindByUserID(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	return &webAuthnUserAdapter{u: u, creds: creds}, nil
}

func (s *WebAuthnService) getUserAdapterByID(ctx context.Context, userIDStr string) (*webAuthnUserAdapter, error) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user id", response.ErrUnauthorized)
	}

	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !u.IsVerified {
		return nil, fmt.Errorf("%w: user email not verified", response.ErrForbidden)
	}

	creds, err := s.creds.FindByUserID(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	return &webAuthnUserAdapter{u: u, creds: creds}, nil
}

func (s *WebAuthnService) saveSession(ctx context.Context, sessionID string, data *webauthn.SessionData) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	// Session valid for 5 minutes
	return s.cache.Set(ctx, "webauthn_session:"+sessionID, b, 5*time.Minute)
}

func (s *WebAuthnService) getSession(ctx context.Context, sessionID string) (*webauthn.SessionData, error) {
	b, err := s.cache.Get(ctx, "webauthn_session:"+sessionID)
	if err != nil {
		// If there is any error, we treat it as an expired or invalid session.
		return nil, fmt.Errorf("%w: webauthn session expired or invalid", response.ErrNotFound)
	}
	var data webauthn.SessionData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	// Once used, delete it to prevent replay attacks
	_ = s.cache.Delete(ctx, "webauthn_session:"+sessionID)
	return &data, nil
}

func (s *WebAuthnService) RegisterBegin(ctx context.Context, cmd port.WebAuthnRegisterBeginCommand) (any, string, error) {
	adapter, err := s.getUserAdapterByID(ctx, cmd.UserID)
	if err != nil {
		return nil, "", err
	}

	creationData, sessionData, err := s.w.BeginRegistration(adapter)
	if err != nil {
		return nil, "", fmt.Errorf("begin registration: %w", err)
	}

	sessionID := uuid.New().String()
	if err := s.saveSession(ctx, sessionID, sessionData); err != nil {
		return nil, "", err
	}

	return creationData, sessionID, nil
}

func (s *WebAuthnService) RegisterFinish(ctx context.Context, cmd port.WebAuthnRegisterFinishCommand) (*port.AuthResponse, error) {
	adapter, err := s.getUserAdapterByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}

	sessionData, err := s.getSession(ctx, cmd.SessionID)
	if err != nil {
		return nil, err
	}

	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(cmd.Body))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid webauthn response", response.ErrInvalidTransition) // or custom validation error
	}

	cred, err := s.w.CreateCredential(adapter, *sessionData, parsedResponse)
	if err != nil {
		return nil, fmt.Errorf("create credential: %w", err)
	}

	// Convert and save
	transports := make([]string, len(cred.Transport))
	for i, t := range cred.Transport {
		transports[i] = string(t)
	}

	domainCred := &domain.WebAuthnCredential{
		ID:                  uuid.New(),
		UserID:              adapter.u.ID,
		CredentialID:        cred.ID,
		PublicKey:           cred.PublicKey,
		AttestationType:     cred.AttestationType,
		Transport:           transports,
		Flags: domain.CredentialFlags{
			UserPresent:    cred.Flags.UserPresent,
			UserVerified:   cred.Flags.UserVerified,
			BackupEligible: cred.Flags.BackupEligible,
			BackupState:    cred.Flags.BackupState,
		},
		AuthenticatorAAGUID: cred.Authenticator.AAGUID,
		SignCount:           cred.Authenticator.SignCount,
		CloneWarning:        cred.Authenticator.CloneWarning,
		CreatedAt:           time.Now().UTC(),
	}

	if err := s.creds.Save(ctx, domainCred); err != nil {
		return nil, err
	}

	accessToken, _, err := s.tokens.IssueAccessToken(adapter.u.ID.String(), adapter.u.Email, adapter.u.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}
	refreshTokenStr, exp, err := s.tokens.IssueRefreshToken(adapter.u.ID.String())
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	h := sha256.New()
	h.Write([]byte(refreshTokenStr))
	rtHash := fmt.Sprintf("%x", h.Sum(nil))

	if err := s.refreshTokens.Save(ctx, &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    adapter.u.ID,
		Hash:      rtHash,
		ExpiresAt: exp,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	avatar := (*string)(nil)
	if adapter.u.AvatarURL != nil {
		avatar = adapter.u.AvatarURL
	}

	return &port.AuthResponse{
		User: port.UserDTO{
			ID:          adapter.u.ID.String(),
			Email:       adapter.u.Email,
			DisplayName: adapter.u.DisplayName,
			AvatarURL:   avatar,
			IsAdmin:     adapter.u.IsAdmin,
			CreatedAt:   adapter.u.CreatedAt.Format(time.RFC3339),
		},
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
	}, nil
}

func (s *WebAuthnService) LoginBegin(ctx context.Context, cmd port.WebAuthnLoginBeginCommand) (any, string, error) {
	adapter, err := s.getUserAdapter(ctx, cmd.Email)
	if err != nil {
		return nil, "", err
	}

	assertionData, sessionData, err := s.w.BeginLogin(adapter)
	if err != nil {
		return nil, "", fmt.Errorf("begin login: %w", err)
	}

	sessionID := uuid.New().String()
	if err := s.saveSession(ctx, sessionID, sessionData); err != nil {
		return nil, "", err
	}

	return assertionData, sessionID, nil
}

func (s *WebAuthnService) LoginFinish(ctx context.Context, cmd port.WebAuthnLoginFinishCommand) (*port.AuthResponse, error) {
	adapter, err := s.getUserAdapter(ctx, cmd.Email)
	if err != nil {
		return nil, err
	}

	sessionData, err := s.getSession(ctx, cmd.SessionID)
	if err != nil {
		return nil, err
	}

	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(cmd.Body))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid webauthn login response", response.ErrInvalidTransition)
	}

	cred, err := s.w.ValidateLogin(adapter, *sessionData, parsedResponse)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid credentials", response.ErrUnauthorized)
	}

	// Find the credential to update sign count
	var matchedCred *domain.WebAuthnCredential
	for _, c := range adapter.creds {
		if bytes.Equal(c.CredentialID, cred.ID) {
			matchedCred = c
			break
		}
	}
	if matchedCred != nil {
		matchedCred.SignCount = cred.Authenticator.SignCount
		matchedCred.CloneWarning = cred.Authenticator.CloneWarning
		now := time.Now().UTC()
		matchedCred.LastUsedAt = &now
		if err := s.creds.Update(ctx, matchedCred); err != nil {
			return nil, err
		}
	}

	accessToken, _, err := s.tokens.IssueAccessToken(adapter.u.ID.String(), adapter.u.Email, adapter.u.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}
	refreshTokenStr, exp, err := s.tokens.IssueRefreshToken(adapter.u.ID.String())
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	h := sha256.New()
	h.Write([]byte(refreshTokenStr))
	rtHash := fmt.Sprintf("%x", h.Sum(nil))

	if err := s.refreshTokens.Save(ctx, &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    adapter.u.ID,
		Hash:      rtHash,
		ExpiresAt: exp,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	avatar := (*string)(nil)
	if adapter.u.AvatarURL != nil {
		avatar = adapter.u.AvatarURL
	}

	return &port.AuthResponse{
		User: port.UserDTO{
			ID:          adapter.u.ID.String(),
			Email:       adapter.u.Email,
			DisplayName: adapter.u.DisplayName,
			AvatarURL:   avatar,
			IsAdmin:     adapter.u.IsAdmin,
			CreatedAt:   adapter.u.CreatedAt.Format(time.RFC3339),
		},
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
	}, nil
}
