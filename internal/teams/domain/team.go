package domain

import (
	"crypto/rand"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrAlreadyOnTeam = errors.New("user already on a team for this event")
	ErrTeamNameTaken = errors.New("team name already taken")
)

type Team struct {
	ID         uuid.UUID
	EventID    uuid.UUID
	Name       string
	InviteCode string
	CreatedBy  uuid.UUID
	IsLocked   bool
	LockedAt   *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type TeamMemberRole string

const (
	RoleOwner  TeamMemberRole = "owner"
	RoleMember TeamMemberRole = "member"
)

type TeamMember struct {
	ID       uuid.UUID
	TeamID   uuid.UUID
	UserID   uuid.UUID
	EventID  uuid.UUID
	Role     TeamMemberRole
	JoinedAt time.Time
}

// GenerateInviteCode generates an 8-character uppercase alphanumeric code.
func GenerateInviteCode() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.Grow(8)
	for i := 0; i < 8; i++ {
		sb.WriteByte(charset[int(b[i])%len(charset)])
	}
	return sb.String(), nil
}
