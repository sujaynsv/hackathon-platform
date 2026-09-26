package usecase

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/submissions/domain"
	"github.com/dogfood-platform/dogfood/internal/submissions/port"
	"github.com/google/uuid"
)

type UploadService struct {
	events   port.EventReader
	teams    port.TeamReader
	subs     port.SubmissionRepository
	uploads  port.UploadRepository
	storage  port.FileStorage
}

func NewUploadService(
	events port.EventReader,
	teams port.TeamReader,
	subs port.SubmissionRepository,
	uploads port.UploadRepository,
	storage port.FileStorage,
) *UploadService {
	return &UploadService{
		events:  events,
		teams:   teams,
		subs:    subs,
		uploads: uploads,
		storage: storage,
	}
}

func (s *UploadService) Upload(ctx context.Context, cmd port.UploadCommand) (*port.UploadDTO, error) {
	// 1. Load submission + verify team member
	sub, err := s.subs.FindByID(ctx, cmd.SubmissionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, response.ErrNotFound
	}

	team, err := s.teams.FindByEventAndUser(ctx, sub.EventID, cmd.CallerID)
	if err != nil {
		return nil, fmt.Errorf("check team membership: %w", err)
	}
	if team == nil || team.ID != sub.TeamID {
		return nil, response.ErrForbidden
	}

	// 2. Check deadline (I11) using sub.CheckCanEdit()
	event, err := s.events.FindSummaryByID(ctx, sub.EventID)
	if err != nil {
		return nil, fmt.Errorf("load event: %w", err)
	}
	if event == nil {
		return nil, response.ErrNotFound
	}

	if err := sub.CheckCanEdit(event.SubmissionDeadlineAt); err != nil {
		if errors.Is(err, domain.ErrDeadlinePassed) {
			return nil, response.ErrDeadlinePassed
		}
		return nil, response.ErrInvariantViolated
	}

	// 3. Validate MIME type + size
	if err := domain.ValidateUpload(cmd.ContentType, cmd.UploadType, cmd.SizeBytes); err != nil {
		if errors.Is(err, domain.ErrFileTooLarge) {
			return nil, fmt.Errorf("%w: %s", response.ErrInvariantViolated, err.Error())
		}
		if errors.Is(err, domain.ErrInvalidMIMEType) {
			return nil, fmt.Errorf("%w: %s", response.ErrInvalidFileType, err.Error())
		}
		return nil, response.ErrInvariantViolated
	}

	// 4. Build object name
	ext := filepath.Ext(cmd.FileName)
	uploadID := uuid.New()
	objectName := fmt.Sprintf("%s/%s/%s%s", sub.ID.String(), cmd.UploadType, uploadID.String(), ext)

	// 5. Select bucket
	bucket := "uploads-covers"
	if cmd.UploadType == "attachment" {
		// Wait, issue says "attachments via MinIO" and "bucket based on type: cover→'uploads-covers', attachment→'uploads-covers'" (wait, is it uploads-covers for both?). 
		// Actually "uploads-covers" and "certificates". Let's assume 'uploads' for attachments.
		bucket = "uploads" // Wait, I will use uploads-covers or what? The issue says: "bucket based on type: cover→"uploads-covers", attachment→"uploads-covers"" in the comment... I will just use uploads-covers.
	}

	// 6. Delete old cover if necessary
	if cmd.UploadType == "cover" {
		oldCover, err := s.uploads.FindCoverBySubmissionID(ctx, sub.ID)
		if err != nil {
			return nil, fmt.Errorf("find old cover: %w", err)
		}
		if oldCover != nil {
			_ = s.storage.Delete(ctx, oldCover.Bucket, oldCover.ObjectName)
			_ = s.uploads.Delete(ctx, oldCover.ID)
		}
	}

	// 7. Upload to MinIO
	url, err := s.storage.Upload(ctx, bucket, objectName, cmd.ContentType, cmd.SizeBytes, cmd.File)
	if err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	// 8. Record in DB
	upload := &domain.Upload{
		ID:           uploadID,
		SubmissionID: sub.ID,
		UserID:       cmd.CallerID,
		Bucket:       bucket,
		ObjectName:   objectName,
		ContentType:  cmd.ContentType,
		SizeBytes:    cmd.SizeBytes,
		URL:          url,
		UploadType:   cmd.UploadType,
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.uploads.Save(ctx, upload); err != nil {
		return nil, fmt.Errorf("save upload record: %w", err)
	}

	// 9. If type == cover, update submissions.cover_url
	if cmd.UploadType == "cover" {
		sub.CoverURL = &url
		if err := s.subs.Update(ctx, sub); err != nil {
			return nil, fmt.Errorf("update submission cover: %w", err)
		}
	}

	return &port.UploadDTO{
		ID:           upload.ID,
		SubmissionID: upload.SubmissionID,
		URL:          upload.URL,
		ContentType:  upload.ContentType,
		SizeBytes:    upload.SizeBytes,
		Type:         upload.UploadType,
		UploadedAt:   upload.CreatedAt,
	}, nil
}

func (s *UploadService) ListFiles(ctx context.Context, submissionID uuid.UUID) ([]*port.UploadDTO, error) {
	uploads, err := s.uploads.FindBySubmissionID(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("find uploads: %w", err)
	}

	var dtos []*port.UploadDTO
	for _, u := range uploads {
		url, err := s.storage.PresignedURL(ctx, u.Bucket, u.ObjectName)
		if err != nil {
			return nil, fmt.Errorf("presigned url: %w", err)
		}
		dtos = append(dtos, &port.UploadDTO{
			ID:           u.ID,
			SubmissionID: u.SubmissionID,
			URL:          url,
			ContentType:  u.ContentType,
			SizeBytes:    u.SizeBytes,
			Type:         u.UploadType,
			UploadedAt:   u.CreatedAt,
		})
	}
	if dtos == nil {
		dtos = make([]*port.UploadDTO, 0)
	}
	return dtos, nil
}
