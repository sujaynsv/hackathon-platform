package usecase

import (
	"context"

	"github.com/dogfood-platform/dogfood/internal/submissions/port"
)

type ListSubmissionsService struct {
	subs   port.SubmissionRepository
	events port.EventReader
}

func NewListSubmissionsService(
	subs port.SubmissionRepository,
	events port.EventReader,
) *ListSubmissionsService {
	return &ListSubmissionsService{
		subs:   subs,
		events: events,
	}
}

func (s *ListSubmissionsService) List(ctx context.Context, query port.ListSubmissionsQuery) ([]*port.SubmissionDTO, int, error) {
	// 1. Get event ID from slug
	event, err := s.events.FindSummaryBySlug(ctx, query.EventSlug)
	if err != nil {
		return nil, 0, err
	}

	// 2. Load submissions
	rows, totalCount, err := s.subs.ListGallery(ctx, event.ID, query.TrackID, query.Page, query.PageSize)
	if err != nil {
		return nil, 0, err
	}

	// 3. Map to DTOs
	var dtos []*port.SubmissionDTO
	for _, row := range rows {
		dto := &port.SubmissionDTO{
			SubmissionID: row.SubmissionID,
			Title:        row.Title,
			Status:       row.Status,
			TeamID:       row.TeamID,
			TeamName:     row.TeamName,
			EventSlug:    query.EventSlug,
			RepoURL:      row.RepoURL,
			DemoURL:      row.DemoURL,
			VideoURL:     row.VideoURL,
			CoverURL:     row.CoverURL,
			CreatedAt:    row.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			UpdatedAt:    row.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if row.TrackID != nil && row.TrackName != nil {
			dto.Track = &port.TrackDTO{
				TrackID: *row.TrackID,
				Name:    *row.TrackName,
			}
		}
		if row.SubmittedAt != nil {
			submittedAtStr := row.SubmittedAt.UTC().Format("2006-01-02T15:04:05Z")
			dto.SubmittedAt = &submittedAtStr
		}
		dtos = append(dtos, dto)
	}

	return dtos, totalCount, nil
}
