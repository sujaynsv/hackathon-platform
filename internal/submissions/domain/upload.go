package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrFileTooLarge = errors.New("file exceeds maximum size of 50MB")
var ErrInvalidMIMEType = errors.New("file type not allowed")

const MaxFileSizeBytes = 50 * 1024 * 1024 // 50MB

var allowedCoverTypes = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/webp": true,
}
var allowedAttachmentTypes = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/webp": true,
	"application/pdf": true, "application/zip": true, "video/mp4": true,
}

type Upload struct {
	ID           uuid.UUID
	SubmissionID uuid.UUID
	UserID       uuid.UUID
	Bucket       string
	ObjectName   string
	ContentType  string
	SizeBytes    int64
	URL          string
	UploadType   string // "cover" or "attachment"
	CreatedAt    time.Time
}

func ValidateUpload(contentType, uploadType string, sizeBytes int64) error {
	if sizeBytes > MaxFileSizeBytes {
		return ErrFileTooLarge
	}
	ct := strings.ToLower(contentType)
	if uploadType == "cover" && !allowedCoverTypes[ct] {
		return ErrInvalidMIMEType
	}
	if uploadType == "attachment" && !allowedAttachmentTypes[ct] {
		return ErrInvalidMIMEType
	}
	return nil
}
