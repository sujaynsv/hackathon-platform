package domain

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	TeamOwnerRole  = "owner"
	TeamMemberRole = "member"
	inviteAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	inviteCodeSize = 8
)

var (
	ErrInvalidTeamName = errors.New("team name is required")
	ErrAlreadyOnTeam   = errors.New("user is already on a team in this event")
	ErrTeamNameTaken   = errors.New("team name already taken in this event")
	ErrInviteCodeTaken = errors.New("invite code already exists")
	ErrEventNotRegOpen = errors.New("team creation only allowed during registration_open")
	ErrNotParticipant  = errors.New("user must be a registered participant to create a team")
)

type Team struct {
	ID         uuid.UUID
	EventID    uuid.UUID
	Name       string
	InviteCode string
	CreatedBy  uuid.UUID
	IsLocked   bool
	Members    []TeamMember
	CreatedAt  time.Time
}

type TeamMember struct {
	TeamID      uuid.UUID
	UserID      uuid.UUID
	EventID     uuid.UUID
	DisplayName string
	AvatarURL   *string
	Role        string
	JoinedAt    time.Time
}

func NewTeam(eventID, creatorID uuid.UUID, name, inviteCode string, now time.Time) (*Team, *TeamMember, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil, ErrInvalidTeamName
	}

	team := &Team{
		ID:         uuid.New(),
		EventID:    eventID,
		Name:       name,
		InviteCode: inviteCode,
		CreatedBy:  creatorID,
		CreatedAt:  now.UTC(),
	}
	leader := &TeamMember{
		TeamID:   team.ID,
		UserID:   creatorID,
		EventID:  eventID,
		Role:     TeamOwnerRole,
		JoinedAt: now.UTC(),
	}
	team.Members = []TeamMember{*leader}
	return team, leader, nil
}

// GenerateInviteCode returns eight unbiased uppercase alphanumeric characters.
func GenerateInviteCode() (string, error) {
	code := make([]byte, 0, inviteCodeSize)
	buffer := make([]byte, inviteCodeSize)
	maxAccepted := byte(256 - (256 % len(inviteAlphabet)))
	for len(code) < inviteCodeSize {
		if _, err := rand.Read(buffer); err != nil {
			return "", fmt.Errorf("generate invite code: %w", err)
		}
		for _, value := range buffer {
			if value >= maxAccepted {
				continue
			}
			code = append(code, inviteAlphabet[int(value)%len(inviteAlphabet)])
			if len(code) == inviteCodeSize {
				break
			}
		}
	}
	return string(code), nil
}
