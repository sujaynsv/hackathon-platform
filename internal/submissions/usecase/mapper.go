package usecase

import (
	"time"

	"github.com/dogfood-platform/dogfood/internal/submissions/domain"
	"github.com/dogfood-platform/dogfood/internal/submissions/port"
)

func toSubmissionDTO(sub *domain.Submission, teamName, eventSlug string, track *port.TrackDTO) *port.SubmissionDTO {
	var submittedAt *string
	if sub.SubmittedAt != nil {
		s := sub.SubmittedAt.Format(time.RFC3339)
		submittedAt = &s
	}

	return &port.SubmissionDTO{
		SubmissionID: sub.ID,
		Title:        sub.Title,
		Status:       string(sub.Status),
		TeamID:       sub.TeamID,
		TeamName:     teamName,
		EventSlug:    eventSlug,
		Track:        track,
		RepoURL:      sub.RepoURL,
		DemoURL:      sub.DemoURL,
		VideoURL:     sub.VideoURL,
		CoverURL:     sub.CoverURL,
		CreatedAt:    sub.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    sub.UpdatedAt.Format(time.RFC3339),
		SubmittedAt:  submittedAt,
	}
}
