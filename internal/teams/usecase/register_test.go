package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/teams/port"
	"github.com/dogfood-platform/dogfood/internal/teams/usecase"
	"github.com/google/uuid"
)

type registrationEventReader struct {
	event *port.EventSummary
	err   error
}

func (r registrationEventReader) FindSummaryBySlug(context.Context, string) (*port.EventSummary, error) {
	return r.event, r.err
}

type registrationParticipants struct {
	roles        map[string]bool
	registeredAt time.Time
	grantErr     error
	revokeErr    error
	revoked      bool
}

func (r *registrationParticipants) HasRole(_ context.Context, _, _ uuid.UUID, role string) (bool, error) {
	return r.roles[role], nil
}

func (r *registrationParticipants) GetRole(_ context.Context, _, _ uuid.UUID) (string, error) {
	if r.roles["participant"] {
		return "participant", nil
	}
	return "unregistered", nil
}

func (r *registrationParticipants) GrantParticipantRole(context.Context, uuid.UUID, uuid.UUID) (time.Time, error) {
	if r.grantErr != nil {
		return time.Time{}, r.grantErr
	}
	r.roles["participant"] = true
	return r.registeredAt, nil
}

func (r *registrationParticipants) RevokeParticipantRole(context.Context, uuid.UUID, uuid.UUID) error {
	if r.revokeErr != nil {
		return r.revokeErr
	}
	r.roles["participant"] = false
	r.revoked = true
	return nil
}

type registrationTeamMembers struct {
	hasTeam bool
	err     error
}

func (r registrationTeamMembers) HasTeamInEvent(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return r.hasTeam, r.err
}

func newRegistrationFixture(status string, roles map[string]bool, hasTeam bool) (*port.EventSummary, *registrationParticipants, registrationTeamMembers) {
	return &port.EventSummary{
		ID:     uuid.New(),
		Slug:   "hackathon-2026",
		Status: status,
	}, &registrationParticipants{
		roles:        roles,
		registeredAt: time.Date(2026, time.September, 14, 8, 0, 0, 0, time.UTC),
	}, registrationTeamMembers{hasTeam: hasTeam}
}

func TestRegisterService_ValidRegistration(t *testing.T) {
	event, participants, _ := newRegistrationFixture("registration_open", map[string]bool{}, false)
	svc := usecase.NewRegisterService(registrationEventReader{event: event}, participants)

	dto, err := svc.Register(context.Background(), port.RegisterCommand{UserID: uuid.New(), EventSlug: event.Slug})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if !dto.Registered || dto.EventID != event.ID.String() || dto.EventSlug != event.Slug || dto.RegisteredAt != "2026-09-14T08:00:00Z" {
		t.Fatalf("unexpected registration DTO: %+v", dto)
	}
}

func TestRegisterService_EventNotOpen(t *testing.T) {
	event, participants, _ := newRegistrationFixture("draft", map[string]bool{}, false)
	svc := usecase.NewRegisterService(registrationEventReader{event: event}, participants)

	_, err := svc.Register(context.Background(), port.RegisterCommand{UserID: uuid.New(), EventSlug: event.Slug})
	if !errors.Is(err, response.ErrInvariantViolated) {
		t.Fatalf("expected ErrInvariantViolated, got %v", err)
	}
}

func TestRegisterService_DeadlinePassed(t *testing.T) {
	closesAt := time.Now().Add(-time.Hour)
	event, participants, _ := newRegistrationFixture("registration_open", map[string]bool{}, false)
	event.RegistrationClosesAt = &closesAt
	svc := usecase.NewRegisterService(registrationEventReader{event: event}, participants)

	_, err := svc.Register(context.Background(), port.RegisterCommand{UserID: uuid.New(), EventSlug: event.Slug})
	if !errors.Is(err, response.ErrDeadlinePassed) {
		t.Fatalf("expected ErrDeadlinePassed, got %v", err)
	}
}

func TestRegisterService_AlreadyRegistered(t *testing.T) {
	event, participants, _ := newRegistrationFixture("registration_open", map[string]bool{"participant": true}, false)
	svc := usecase.NewRegisterService(registrationEventReader{event: event}, participants)

	_, err := svc.Register(context.Background(), port.RegisterCommand{UserID: uuid.New(), EventSlug: event.Slug})
	if !errors.Is(err, response.ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestRegisterService_UserIsJudge(t *testing.T) {
	event, participants, _ := newRegistrationFixture("registration_open", map[string]bool{"judge": true}, false)
	svc := usecase.NewRegisterService(registrationEventReader{event: event}, participants)

	_, err := svc.Register(context.Background(), port.RegisterCommand{UserID: uuid.New(), EventSlug: event.Slug})
	if !errors.Is(err, response.ErrInvariantViolated) {
		t.Fatalf("expected ErrInvariantViolated, got %v", err)
	}
}

func TestUnregisterService_UserHasTeam(t *testing.T) {
	event, participants, teams := newRegistrationFixture("registration_open", map[string]bool{"participant": true}, true)
	svc := usecase.NewUnregisterService(registrationEventReader{event: event}, participants, teams)

	err := svc.Unregister(context.Background(), port.UnregisterCommand{UserID: uuid.New(), EventSlug: event.Slug})
	if !errors.Is(err, response.ErrInvariantViolated) {
		t.Fatalf("expected ErrInvariantViolated, got %v", err)
	}
	if participants.revoked {
		t.Fatal("participant role was revoked despite an existing team")
	}
}

func TestUnregisterService_UserNotRegistered(t *testing.T) {
	event, participants, teams := newRegistrationFixture("registration_open", map[string]bool{}, false)
	svc := usecase.NewUnregisterService(registrationEventReader{event: event}, participants, teams)

	err := svc.Unregister(context.Background(), port.UnregisterCommand{UserID: uuid.New(), EventSlug: event.Slug})
	if !errors.Is(err, response.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUnregisterService_NoTeamRemovesRole(t *testing.T) {
	event, participants, teams := newRegistrationFixture("registration_open", map[string]bool{"participant": true}, false)
	svc := usecase.NewUnregisterService(registrationEventReader{event: event}, participants, teams)

	err := svc.Unregister(context.Background(), port.UnregisterCommand{UserID: uuid.New(), EventSlug: event.Slug})
	if err != nil {
		t.Fatalf("Unregister returned error: %v", err)
	}
	if !participants.revoked {
		t.Fatal("participant role was not revoked")
	}
}
