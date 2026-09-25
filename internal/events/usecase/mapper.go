package usecase

import (
	"github.com/dogfood-platform/dogfood/internal/events/domain"
	"github.com/dogfood-platform/dogfood/internal/events/port"
)

func toEventDTO(e *domain.Event) port.EventDTO {
	return port.EventDTO{
		ID:                   e.ID,
		Slug:                 e.Slug,
		Title:                e.Title,
		Description:          e.Description,
		BannerURL:            e.BannerURL,
		OrganizerID:          e.OrganizerID,
		Status:               string(e.Status),
		MaxTeamSize:          e.MaxTeamSize,
		RegistrationOpensAt:  e.RegistrationOpensAt,
		RegistrationClosesAt: e.RegistrationClosesAt,
		SubmissionDeadlineAt: e.SubmissionDeadlineAt,
		JudgingDeadlineAt:    e.JudgingDeadlineAt,
		VotingOpensAt:        e.VotingOpensAt,
		VotingClosesAt:       e.VotingClosesAt,
		CreatedAt:            e.CreatedAt,
		UpdatedAt:            e.UpdatedAt,
	}
}

func toTrackDTO(t *domain.Track) port.TrackDTO {
	return port.TrackDTO{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
	}
}
