package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/dogfood-platform/dogfood/internal/submissions/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type PgUploadRepository struct {
	db *sqlx.DB
}

func NewPgUploadRepository(db *sqlx.DB) *PgUploadRepository {
	return &PgUploadRepository{db: db}
}

func (r *PgUploadRepository) Save(ctx context.Context, upload *domain.Upload) error {
	const q = `
		INSERT INTO uploads (
			id, submission_id, user_id, bucket, object_name, 
			content_type, size_bytes, url, upload_type, created_at
		) VALUES (
			:id, :submission_id, :user_id, :bucket, :object_name, 
			:content_type, :size_bytes, :url, :upload_type, :created_at
		)
	`
	_, err := r.db.NamedExecContext(ctx, q, map[string]interface{}{
		"id":            upload.ID,
		"submission_id": upload.SubmissionID,
		"user_id":       upload.UserID,
		"bucket":        upload.Bucket,
		"object_name":   upload.ObjectName,
		"content_type":  upload.ContentType,
		"size_bytes":    upload.SizeBytes,
		"url":           upload.URL,
		"upload_type":   upload.UploadType,
		"created_at":    upload.CreatedAt,
	})
	return err
}

type uploadRow struct {
	ID           uuid.UUID `db:"id"`
	SubmissionID uuid.UUID `db:"submission_id"`
	UserID       uuid.UUID `db:"user_id"`
	Bucket       string    `db:"bucket"`
	ObjectName   string    `db:"object_name"`
	ContentType  string    `db:"content_type"`
	SizeBytes    int64     `db:"size_bytes"`
	URL          string    `db:"url"`
	UploadType   string    `db:"upload_type"`
	CreatedAt    time.Time `db:"created_at"`
}

func toDomainUpload(r uploadRow) *domain.Upload {
	return &domain.Upload{
		ID:           r.ID,
		SubmissionID: r.SubmissionID,
		UserID:       r.UserID,
		Bucket:       r.Bucket,
		ObjectName:   r.ObjectName,
		ContentType:  r.ContentType,
		SizeBytes:    r.SizeBytes,
		URL:          r.URL,
		UploadType:   r.UploadType,
		CreatedAt:    r.CreatedAt,
	}
}

func (r *PgUploadRepository) FindBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]*domain.Upload, error) {
	const q = `
		SELECT 
			id, submission_id, user_id, bucket, object_name, 
			content_type, size_bytes, url, upload_type, created_at
		FROM uploads
		WHERE submission_id = $1
		ORDER BY created_at DESC
	`
	var rows []uploadRow
	err := r.db.SelectContext(ctx, &rows, q, submissionID)
	if err != nil {
		return nil, err
	}

	var result []*domain.Upload
	for _, row := range rows {
		result = append(result, toDomainUpload(row))
	}
	return result, nil
}

func (r *PgUploadRepository) FindCoverBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*domain.Upload, error) {
	const q = `
		SELECT 
			id, submission_id, user_id, bucket, object_name, 
			content_type, size_bytes, url, upload_type, created_at
		FROM uploads
		WHERE submission_id = $1 AND upload_type = 'cover'
		ORDER BY created_at DESC
		LIMIT 1
	`
	var row uploadRow
	err := r.db.GetContext(ctx, &row, q, submissionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toDomainUpload(row), nil
}

func (r *PgUploadRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM uploads WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}
