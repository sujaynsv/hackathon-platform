package email

import (
	"context"
	"log/slog"
)

// StubSender is a no-op email sender for development and testing.
type StubSender struct {
	logger *slog.Logger
}

func NewStubSender(logger *slog.Logger) *StubSender {
	return &StubSender{logger: logger}
}

func (s *StubSender) SendVerificationEmail(ctx context.Context, email, token string) error {
	verifyURL := "http://localhost:3000/verify-email?token=" + token
	s.logger.Info("StubSender: verification email generated", "email", email, "verify_url", verifyURL, "token", token)
	return nil
}
