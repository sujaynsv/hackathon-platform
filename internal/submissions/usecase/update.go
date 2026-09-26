package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/submissions/domain"
	"github.com/dogfood-platform/dogfood/internal/submissions/port"
)

type UpdateSubmissionService struct {
	events port.EventReader
	teams  port.TeamReader
	tracks port.TrackReader
	subs   port.SubmissionRepository
}

func NewUpdateSubmissionService(events port.EventReader, teams port.TeamReader, tracks port.TrackReader, subs port.SubmissionRepository) *UpdateSubmissionService {
	return &UpdateSubmissionService{events: events, teams: teams, tracks: tracks, subs: subs}
}

func (s *UpdateSubmissionService) Update(ctx context.Context, cmd port.UpdateSubmissionCommand) (*port.SubmissionDTO, error) {
	// 1. Load submission
	sub, err := s.subs.FindByID(ctx, cmd.SubmissionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, response.ErrNotFound
	}

	// 2. Verify caller is team member
	team, err := s.teams.FindByEventAndUser(ctx, sub.EventID, cmd.CallerID)
	if err != nil || team == nil || team.ID != sub.TeamID {
		return nil, fmt.Errorf("%w: not a member of this team", response.ErrForbidden)
	}

	// 3. Load event deadline
	event, _ := s.events.FindSummaryByID(ctx, sub.EventID)

	// 4. I11 + I13: check can edit
	if err := sub.CheckCanEdit(event.SubmissionDeadlineAt); err != nil {
		if errors.Is(err, domain.ErrDeadlinePassed) {
			return nil, fmt.Errorf("%w: %s", response.ErrDeadlinePassed, err.Error())
		}
		return nil, fmt.Errorf("%w: %s", response.ErrInvariantViolated, err.Error())
	}

	// 5. Apply updates
	if cmd.Title != nil {
		sub.Title = *cmd.Title
	}
	if cmd.Description != nil {
		sub.Description = cmd.Description
	}
	if cmd.RepoURL != nil {
		sub.RepoURL = cmd.RepoURL
	}
	if cmd.DemoURL != nil {
		sub.DemoURL = cmd.DemoURL
	}
	sub.UpdatedAt = time.Now().UTC()

	if err := s.subs.Update(ctx, sub); err != nil {
		return nil, err
	}

	var trackDTO *port.TrackDTO
	if sub.TrackID != nil {
		tracks, _ := s.tracks.FindByEventID(ctx, event.ID)
		for _, t := range tracks {
			if t.ID == *sub.TrackID {
				trackDTO = &port.TrackDTO{TrackID: t.ID, Name: t.Name}
				break
			}
		}
	}

	return toSubmissionDTO(sub, team.Name, event.Slug, trackDTO), nil
}
