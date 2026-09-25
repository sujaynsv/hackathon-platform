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
	redacted := "[REDACTED]"
	if len(token) > 8 {
		redacted = token[:4] + "..." + token[len(token)-4:]
	}
	s.logger.Info("StubSender: would send verification email", "email", email, "token", redacted)
	return nil
}
