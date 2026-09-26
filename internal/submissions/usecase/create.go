package usecase

import (
	"context"
	"fmt"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/submissions/domain"
	"github.com/dogfood-platform/dogfood/internal/submissions/port"
)

type CreateSubmissionService struct {
	events port.EventReader
	teams  port.TeamReader
	tracks port.TrackReader
	subs   port.SubmissionRepository
}

func NewCreateSubmissionService(
	events port.EventReader,
	teams port.TeamReader,
	tracks port.TrackReader,
	subs port.SubmissionRepository,
) *CreateSubmissionService {
	return &CreateSubmissionService{
		events: events,
		teams:  teams,
		tracks: tracks,
		subs:   subs,
	}
}

func (s *CreateSubmissionService) Create(ctx context.Context, cmd port.CreateSubmissionCommand) (*port.SubmissionDTO, error) {
	// 1. Load event
	event, err := s.events.FindSummaryBySlug(ctx, cmd.EventSlug)
	if err != nil {
		return nil, err
	}

	// 2. I9: event must be submissions_open
	if err := domain.CheckCanCreate(event.Status); err != nil {
		return nil, fmt.Errorf("%w: %s", response.ErrInvariantViolated, err.Error())
	}

	// 3. Find caller's team — 403 if they have no team
	team, err := s.teams.FindByEventAndUser(ctx, event.ID, cmd.CallerID)
	if err != nil || team == nil {
		return nil, fmt.Errorf("%w: must be a team member to submit", response.ErrForbidden)
	}

	// 4. I2: check team doesn't already have a submission
	existing, _ := s.subs.FindByTeamAndEvent(ctx, team.ID, event.ID)
	if existing != nil {
		return nil, fmt.Errorf("%w: team already has a submission", response.ErrInvariantViolated)
	}

	// 5. Validate trackID belongs to this event (if provided)
	var matchedTrack *port.TrackSummary
	if cmd.TrackID != nil {
		tracks, _ := s.tracks.FindByEventID(ctx, event.ID)
		for _, t := range tracks {
			if t.ID == *cmd.TrackID {
				matchedTrack = t
				break
			}
		}
		if matchedTrack == nil {
			return nil, fmt.Errorf("trackId does not belong to this event")
		}
	}

	// 6. Create draft submission
	sub, err := domain.NewDraftSubmission(team.ID, event.ID, cmd.TrackID, cmd.Title)
	if err != nil {
		return nil, err
	}
	sub.Description = cmd.Description
	sub.RepoURL = cmd.RepoURL
	sub.DemoURL = cmd.DemoURL
	sub.VideoURL = cmd.VideoURL

	if err := s.subs.Save(ctx, sub); err != nil {
		return nil, err
	}

	var trackDTO *port.TrackDTO
	if matchedTrack != nil {
		trackDTO = &port.TrackDTO{
			TrackID: matchedTrack.ID,
			Name:    matchedTrack.Name,
		}
	}

	return toSubmissionDTO(sub, team.Name, event.Slug, trackDTO), nil
}
