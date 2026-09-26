package usecase

import (
	"time"

	"github.com/dogfood-platform/dogfood/internal/teams/domain"
	"github.com/dogfood-platform/dogfood/internal/teams/port"
)

func toTeamDTO(team *domain.Team) *port.TeamDTO {
	members := make([]port.TeamMemberDTO, len(team.Members))
	for i, member := range team.Members {
		members[i] = port.TeamMemberDTO{
			UserID:      member.UserID.String(),
			DisplayName: member.DisplayName,
			AvatarURL:   member.AvatarURL,
			Role:        member.Role,
			JoinedAt:    member.JoinedAt.UTC().Format(time.RFC3339),
		}
	}

	return &port.TeamDTO{
		ID:         team.ID.String(),
		EventID:    team.EventID.String(),
		Name:       team.Name,
		InviteCode: team.InviteCode,
		Members:    members,
		CreatedAt:  team.CreatedAt.UTC().Format(time.RFC3339),
	}
}
