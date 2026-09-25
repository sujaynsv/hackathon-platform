package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/submissions/domain"
	"github.com/dogfood-platform/dogfood/internal/submissions/port"
	"github.com/dogfood-platform/dogfood/internal/submissions/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockEventReader struct {
	mock.Mock
}

func (m *mockEventReader) FindSummaryBySlug(ctx context.Context, slug string) (*port.EventSummary, error) {
	args := m.Called(ctx, slug)
	var res *port.EventSummary
	if args.Get(0) != nil {
		res = args.Get(0).(*port.EventSummary)
	}
	return res, args.Error(1)
}

type mockTeamReader struct {
	mock.Mock
}

func (m *mockTeamReader) FindByEventAndUser(ctx context.Context, eventID, userID uuid.UUID) (*port.TeamSummary, error) {
	args := m.Called(ctx, eventID, userID)
	var res *port.TeamSummary
	if args.Get(0) != nil {
		res = args.Get(0).(*port.TeamSummary)
	}
	return res, args.Error(1)
}

type mockTrackReader struct {
	mock.Mock
}

func (m *mockTrackReader) FindByEventID(ctx context.Context, eventID uuid.UUID) ([]*port.TrackSummary, error) {
	args := m.Called(ctx, eventID)
	var res []*port.TrackSummary
	if args.Get(0) != nil {
		res = args.Get(0).([]*port.TrackSummary)
	}
	return res, args.Error(1)
}

type mockSubmissionRepo struct {
	mock.Mock
}

func (m *mockSubmissionRepo) Save(ctx context.Context, sub *domain.Submission) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *mockSubmissionRepo) FindByTeamAndEvent(ctx context.Context, teamID, eventID uuid.UUID) (*domain.Submission, error) {
	args := m.Called(ctx, teamID, eventID)
	var res *domain.Submission
	if args.Get(0) != nil {
		res = args.Get(0).(*domain.Submission)
	}
	return res, args.Error(1)
}

func (m *mockSubmissionRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Submission, error) {
	args := m.Called(ctx, id)
	var res *domain.Submission
	if args.Get(0) != nil {
		res = args.Get(0).(*domain.Submission)
	}
	return res, args.Error(1)
}

func (m *mockSubmissionRepo) Update(ctx context.Context, sub *domain.Submission) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func TestCreateSubmission_Success(t *testing.T) {
	events := new(mockEventReader)
	teams := new(mockTeamReader)
	tracks := new(mockTrackReader)
	subs := new(mockSubmissionRepo)
	svc := usecase.NewCreateSubmissionService(events, teams, tracks, subs)

	eventID := uuid.New()
	teamID := uuid.New()
	callerID := uuid.New()
	slug := "test-event"

	events.On("FindSummaryBySlug", mock.Anything, slug).Return(&port.EventSummary{
		ID:     eventID,
		Status: "submissions_open",
		Slug:   slug,
	}, nil)

	teams.On("FindByEventAndUser", mock.Anything, eventID, callerID).Return(&port.TeamSummary{
		ID:   teamID,
		Name: "Test Team",
	}, nil)

	subs.On("FindByTeamAndEvent", mock.Anything, teamID, eventID).Return(nil, nil)

	subs.On("Save", mock.Anything, mock.MatchedBy(func(s *domain.Submission) bool {
		return s.Title == "Test Project" && s.TeamID == teamID && s.EventID == eventID
	})).Return(nil)

	cmd := port.CreateSubmissionCommand{
		CallerID:  callerID,
		EventSlug: slug,
		Title:     "Test Project",
	}

	dto, err := svc.Create(context.Background(), cmd)
	assert.NoError(t, err)
	assert.NotNil(t, dto)
	assert.Equal(t, "Test Project", dto.Title)
	assert.Equal(t, "draft", dto.Status)

	events.AssertExpectations(t)
	teams.AssertExpectations(t)
	subs.AssertExpectations(t)
}

func TestCreateSubmission_NotSubmissionsOpen(t *testing.T) {
	events := new(mockEventReader)
	teams := new(mockTeamReader)
	tracks := new(mockTrackReader)
	subs := new(mockSubmissionRepo)
	svc := usecase.NewCreateSubmissionService(events, teams, tracks, subs)

	eventID := uuid.New()
	slug := "test-event"

	events.On("FindSummaryBySlug", mock.Anything, slug).Return(&port.EventSummary{
		ID:     eventID,
		Status: "judging",
		Slug:   slug,
	}, nil)

	cmd := port.CreateSubmissionCommand{
		CallerID:  uuid.New(),
		EventSlug: slug,
		Title:     "Test Project",
	}

	dto, err := svc.Create(context.Background(), cmd)
	assert.Error(t, err)
	assert.Nil(t, dto)
	assert.True(t, errors.Is(err, response.ErrInvariantViolated))

	events.AssertExpectations(t)
}

func TestCreateSubmission_AlreadySubmitted(t *testing.T) {
	events := new(mockEventReader)
	teams := new(mockTeamReader)
	tracks := new(mockTrackReader)
	subs := new(mockSubmissionRepo)
	svc := usecase.NewCreateSubmissionService(events, teams, tracks, subs)

	eventID := uuid.New()
	teamID := uuid.New()
	callerID := uuid.New()
	slug := "test-event"

	events.On("FindSummaryBySlug", mock.Anything, slug).Return(&port.EventSummary{
		ID:     eventID,
		Status: "submissions_open",
		Slug:   slug,
	}, nil)

	teams.On("FindByEventAndUser", mock.Anything, eventID, callerID).Return(&port.TeamSummary{
		ID:   teamID,
		Name: "Test Team",
	}, nil)

	subs.On("FindByTeamAndEvent", mock.Anything, teamID, eventID).Return(&domain.Submission{ID: uuid.New()}, nil)

	cmd := port.CreateSubmissionCommand{
		CallerID:  callerID,
		EventSlug: slug,
		Title:     "Test Project",
	}

	dto, err := svc.Create(context.Background(), cmd)
	assert.Error(t, err)
	assert.Nil(t, dto)
	assert.True(t, errors.Is(err, response.ErrInvariantViolated))

	events.AssertExpectations(t)
	teams.AssertExpectations(t)
	subs.AssertExpectations(t)
}
