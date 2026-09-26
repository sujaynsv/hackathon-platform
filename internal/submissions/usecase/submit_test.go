package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/submissions/domain"
	"github.com/dogfood-platform/dogfood/internal/submissions/port"
	"github.com/dogfood-platform/dogfood/internal/submissions/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFinalSubmit_DraftBeforeDeadline_SetsSubmitted(t *testing.T) {
	events := new(mockEventReader)
	teams := new(mockTeamReader)
	tracks := new(mockTrackReader)
	subs := new(mockSubmissionRepo)
	svc := usecase.NewFinalSubmitService(events, teams, tracks, subs)

	subID := uuid.New()
	eventID := uuid.New()
	teamID := uuid.New()
	callerID := uuid.New()

	sub, _ := domain.NewDraftSubmission(teamID, eventID, nil, "Old Title")
	sub.ID = subID

	subs.On("FindByID", mock.Anything, subID).Return(sub, nil)
	teams.On("FindByEventAndUser", mock.Anything, eventID, callerID).Return(&port.TeamSummary{
		ID:   teamID,
		Name: "Test Team",
	}, nil)

	future := time.Now().UTC().Add(time.Hour)
	events.On("FindSummaryByID", mock.Anything, eventID).Return(&port.EventSummary{
		ID:                   eventID,
		Slug:                 "test-event",
		SubmissionDeadlineAt: &future,
	}, nil)

	subs.On("Update", mock.Anything, mock.MatchedBy(func(s *domain.Submission) bool {
		return s.Status == domain.StatusSubmitted && s.SubmittedAt != nil
	})).Return(nil)

	cmd := port.SubmitCommand{
		CallerID:     callerID,
		SubmissionID: subID,
	}

	dto, err := svc.Submit(context.Background(), cmd)
	assert.NoError(t, err)
	assert.Equal(t, "submitted", dto.Status)
	assert.NotNil(t, dto.SubmittedAt)
}

func TestFinalSubmit_AlreadySubmitted_Returns409(t *testing.T) {
	events := new(mockEventReader)
	teams := new(mockTeamReader)
	tracks := new(mockTrackReader)
	subs := new(mockSubmissionRepo)
	svc := usecase.NewFinalSubmitService(events, teams, tracks, subs)

	subID := uuid.New()
	eventID := uuid.New()
	teamID := uuid.New()
	callerID := uuid.New()

	sub, _ := domain.NewDraftSubmission(teamID, eventID, nil, "Old Title")
	sub.ID = subID
	sub.Status = domain.StatusSubmitted

	subs.On("FindByID", mock.Anything, subID).Return(sub, nil)
	teams.On("FindByEventAndUser", mock.Anything, eventID, callerID).Return(&port.TeamSummary{
		ID:   teamID,
		Name: "Test Team",
	}, nil)

	future := time.Now().UTC().Add(time.Hour)
	events.On("FindSummaryByID", mock.Anything, eventID).Return(&port.EventSummary{
		ID:                   eventID,
		SubmissionDeadlineAt: &future,
	}, nil)

	cmd := port.SubmitCommand{
		CallerID:     callerID,
		SubmissionID: subID,
	}

	_, err := svc.Submit(context.Background(), cmd)
	assert.ErrorIs(t, err, response.ErrInvariantViolated)
}

func TestFinalSubmit_PastDeadline_Returns422(t *testing.T) {
	events := new(mockEventReader)
	teams := new(mockTeamReader)
	tracks := new(mockTrackReader)
	subs := new(mockSubmissionRepo)
	svc := usecase.NewFinalSubmitService(events, teams, tracks, subs)

	subID := uuid.New()
	eventID := uuid.New()
	teamID := uuid.New()
	callerID := uuid.New()

	sub, _ := domain.NewDraftSubmission(teamID, eventID, nil, "Old Title")
	sub.ID = subID

	subs.On("FindByID", mock.Anything, subID).Return(sub, nil)
	teams.On("FindByEventAndUser", mock.Anything, eventID, callerID).Return(&port.TeamSummary{
		ID:   teamID,
		Name: "Test Team",
	}, nil)

	past := time.Now().UTC().Add(-time.Hour)
	events.On("FindSummaryByID", mock.Anything, eventID).Return(&port.EventSummary{
		ID:                   eventID,
		SubmissionDeadlineAt: &past,
	}, nil)

	cmd := port.SubmitCommand{
		CallerID:     callerID,
		SubmissionID: subID,
	}

	_, err := svc.Submit(context.Background(), cmd)
	assert.ErrorIs(t, err, response.ErrDeadlinePassed)
}
