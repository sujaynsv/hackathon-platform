package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/submissions/domain"
	"github.com/dogfood-platform/dogfood/internal/submissions/port"
)

type FinalSubmitService struct {
	events port.EventReader
	teams  port.TeamReader
	tracks port.TrackReader
	subs   port.SubmissionRepository
}

func NewFinalSubmitService(events port.EventReader, teams port.TeamReader, tracks port.TrackReader, subs port.SubmissionRepository) *FinalSubmitService {
	return &FinalSubmitService{events: events, teams: teams, tracks: tracks, subs: subs}
}

func (s *FinalSubmitService) Submit(ctx context.Context, cmd port.SubmitCommand) (*port.SubmissionDTO, error) {
	sub, err := s.subs.FindByID(ctx, cmd.SubmissionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, response.ErrNotFound
	}

	// Verify team membership
	team, _ := s.teams.FindByEventAndUser(ctx, sub.EventID, cmd.CallerID)
	if team == nil || team.ID != sub.TeamID {
		return nil, fmt.Errorf("%w: not a team member", response.ErrForbidden)
	}

	event, _ := s.events.FindSummaryByID(ctx, sub.EventID)

	// I11 + I13: use domain method — it handles both checks atomically
	if err := sub.Submit(event.SubmissionDeadlineAt); err != nil {
		if errors.Is(err, domain.ErrDeadlinePassed) {
			return nil, fmt.Errorf("%w", response.ErrDeadlinePassed)
		}
		if errors.Is(err, domain.ErrNotDraft) {
			return nil, fmt.Errorf("%w: submission already finalized", response.ErrInvariantViolated)
		}
		return nil, err
	}

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
