package domain

import (
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGenerateInviteCode_ReturnsEightUppercaseAlphanumericCharacters(t *testing.T) {
	code, err := GenerateInviteCode()
	if err != nil {
		t.Fatalf("GenerateInviteCode returned error: %v", err)
	}
	if matched, _ := regexp.MatchString(`^[A-Z0-9]{8}$`, code); !matched {
		t.Fatalf("invite code %q is not eight uppercase alphanumeric characters", code)
	}
}

func TestGenerateInviteCode_IsUniqueAcrossOneThousandCalls(t *testing.T) {
	codes := make(map[string]struct{}, 1000)
	for range 1000 {
		code, err := GenerateInviteCode()
		if err != nil {
			t.Fatalf("GenerateInviteCode returned error: %v", err)
		}
		if _, exists := codes[code]; exists {
			t.Fatalf("duplicate invite code generated: %s", code)
		}
		codes[code] = struct{}{}
	}
}

func TestNewTeam_CreatesOwnerMemberAndTrimsName(t *testing.T) {
	eventID := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC()

	team, owner, err := NewTeam(eventID, userID, "  Team Rocket  ", "HACK2026", now)
	if err != nil {
		t.Fatalf("NewTeam returned error: %v", err)
	}
	if team.Name != "Team Rocket" {
		t.Fatalf("team name = %q, want trimmed name", team.Name)
	}
	if owner.TeamID != team.ID || owner.UserID != userID || owner.EventID != eventID || owner.Role != TeamOwnerRole {
		t.Fatalf("unexpected owner member: %+v", owner)
	}
	if !owner.JoinedAt.Equal(now) || !team.CreatedAt.Equal(now) {
		t.Fatalf("team/member timestamps do not match supplied time")
	}
}

func TestNewTeam_RejectsBlankName(t *testing.T) {
	_, _, err := NewTeam(uuid.New(), uuid.New(), "  ", "HACK2026", time.Now())
	if err != ErrInvalidTeamName {
		t.Fatalf("error = %v, want ErrInvalidTeamName", err)
	}
}
