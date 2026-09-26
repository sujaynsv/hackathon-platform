package usecase_test

import (
	"context"
	"io"
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

type mockUploadRepo struct {
	mock.Mock
}

func (m *mockUploadRepo) Save(ctx context.Context, upload *domain.Upload) error {
	args := m.Called(ctx, upload)
	return args.Error(0)
}

func (m *mockUploadRepo) FindBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]*domain.Upload, error) {
	args := m.Called(ctx, submissionID)
	var res []*domain.Upload
	if args.Get(0) != nil {
		res = args.Get(0).([]*domain.Upload)
	}
	return res, args.Error(1)
}

func (m *mockUploadRepo) FindCoverBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*domain.Upload, error) {
	args := m.Called(ctx, submissionID)
	var res *domain.Upload
	if args.Get(0) != nil {
		res = args.Get(0).(*domain.Upload)
	}
	return res, args.Error(1)
}

func (m *mockUploadRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockFileStorage struct {
	mock.Mock
}

func (m *mockFileStorage) Upload(ctx context.Context, bucket, objectName, contentType string, size int64, reader io.Reader) (string, error) {
	args := m.Called(ctx, bucket, objectName, contentType, size, reader)
	return args.String(0), args.Error(1)
}

func (m *mockFileStorage) PresignedURL(ctx context.Context, bucket, objectName string) (string, error) {
	args := m.Called(ctx, bucket, objectName)
	return args.String(0), args.Error(1)
}

func (m *mockFileStorage) Delete(ctx context.Context, bucket, objectName string) error {
	args := m.Called(ctx, bucket, objectName)
	return args.Error(0)
}

func TestUploadService_ValidCover_Uploads_RecordsMetadata(t *testing.T) {
	events := new(mockEventReader)
	teams := new(mockTeamReader)
	subs := new(mockSubmissionRepo)
	uploads := new(mockUploadRepo)
	storage := new(mockFileStorage)
	svc := usecase.NewUploadService(events, teams, subs, uploads, storage)

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
		SubmissionDeadlineAt: &future,
	}, nil)

	uploads.On("FindCoverBySubmissionID", mock.Anything, subID).Return((*domain.Upload)(nil), nil)
	storage.On("Upload", mock.Anything, "uploads-covers", mock.Anything, "image/jpeg", int64(100), mock.Anything).Return("https://minio/uploads-covers/...", nil)
	uploads.On("Save", mock.Anything, mock.Anything).Return(nil)
	subs.On("Update", mock.Anything, mock.Anything).Return(nil)

	cmd := port.UploadCommand{
		CallerID:     callerID,
		SubmissionID: subID,
		FileName:     "test.jpg",
		ContentType:  "image/jpeg",
		SizeBytes:    100,
		UploadType:   "cover",
		File:         nil,
	}

	dto, err := svc.Upload(context.Background(), cmd)
	assert.NoError(t, err)
	assert.NotNil(t, dto)
	assert.Equal(t, "https://minio/uploads-covers/...", dto.URL)
	assert.Equal(t, "cover", dto.Type)
}

func TestUploadService_PastDeadline_Returns422(t *testing.T) {
	events := new(mockEventReader)
	teams := new(mockTeamReader)
	subs := new(mockSubmissionRepo)
	uploads := new(mockUploadRepo)
	storage := new(mockFileStorage)
	svc := usecase.NewUploadService(events, teams, subs, uploads, storage)

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

	cmd := port.UploadCommand{
		CallerID:     callerID,
		SubmissionID: subID,
		FileName:     "test.jpg",
		ContentType:  "image/jpeg",
		SizeBytes:    100,
		UploadType:   "cover",
		File:         nil,
	}

	_, err := svc.Upload(context.Background(), cmd)
	assert.ErrorIs(t, err, response.ErrDeadlinePassed)
}

func TestUploadService_NotTeamMember_Returns403(t *testing.T) {
	events := new(mockEventReader)
	teams := new(mockTeamReader)
	subs := new(mockSubmissionRepo)
	uploads := new(mockUploadRepo)
	storage := new(mockFileStorage)
	svc := usecase.NewUploadService(events, teams, subs, uploads, storage)

	subID := uuid.New()
	eventID := uuid.New()
	teamID := uuid.New()
	callerID := uuid.New()

	sub, _ := domain.NewDraftSubmission(teamID, eventID, nil, "Old Title")
	sub.ID = subID

	subs.On("FindByID", mock.Anything, subID).Return(sub, nil)
	teams.On("FindByEventAndUser", mock.Anything, eventID, callerID).Return((*port.TeamSummary)(nil), nil)

	cmd := port.UploadCommand{
		CallerID:     callerID,
		SubmissionID: subID,
	}

	_, err := svc.Upload(context.Background(), cmd)
	assert.ErrorIs(t, err, response.ErrForbidden)
}

func TestUploadService_ReplaceCover_DeletesOldFile(t *testing.T) {
	events := new(mockEventReader)
	teams := new(mockTeamReader)
	subs := new(mockSubmissionRepo)
	uploads := new(mockUploadRepo)
	storage := new(mockFileStorage)
	svc := usecase.NewUploadService(events, teams, subs, uploads, storage)

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
		SubmissionDeadlineAt: &future,
	}, nil)

	oldUpload := &domain.Upload{ID: uuid.New(), Bucket: "uploads-covers", ObjectName: "old"}
	uploads.On("FindCoverBySubmissionID", mock.Anything, subID).Return(oldUpload, nil)
	storage.On("Delete", mock.Anything, "uploads-covers", "old").Return(nil)
	uploads.On("Delete", mock.Anything, oldUpload.ID).Return(nil)
	storage.On("Upload", mock.Anything, "uploads-covers", mock.Anything, "image/jpeg", int64(100), mock.Anything).Return("new_url", nil)
	uploads.On("Save", mock.Anything, mock.Anything).Return(nil)
	subs.On("Update", mock.Anything, mock.Anything).Return(nil)

	cmd := port.UploadCommand{
		CallerID:     callerID,
		SubmissionID: subID,
		FileName:     "test.jpg",
		ContentType:  "image/jpeg",
		SizeBytes:    100,
		UploadType:   "cover",
		File:         nil,
	}

	dto, err := svc.Upload(context.Background(), cmd)
	assert.NoError(t, err)
	assert.Equal(t, "new_url", dto.URL)
}
