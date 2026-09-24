---
id: S-003
title: File Upload — Cover Image + Attachments via MinIO
epic: submissions
owner: Keerthika (backend)
status: "[ ] not-started"
branch: story/S-003-file-upload
blocks: S-004
blocked-by: S-002
---

# S-003 · File Upload — Cover Image + Attachments via MinIO

## Context (Read ALL of these before writing any code)
- `.agents/rules/engineering-standards.md` — Go hexagonal rules; FileStorage port in shared/port/storage.go
- `MASTER-CONTEXT.md` — MinIO SDK (minio-go v7); buckets: `uploads-avatars`, `certificates`, `uploads-covers`, `uploads-banners`; max file size 50MB
- `docs/api-design.md §Submissions` — POST /submissions/{id}/upload, GET /submissions/{id}/files
- `docs/data-model.md §uploads` — uploads table: bucket, object_name, content_type, size_bytes, url
- `docs/invariants.md §I11` — uploads also blocked after deadline

## What We're Building

Team members can upload a cover image and attachments to their submission. Files are stored in MinIO object storage, and metadata (bucket, object name, URL, size) is recorded in the `uploads` table. Uploaded files are linked to the submission. Only team members can upload to their own submission. The submission deadline check from I11 also applies here.

**Endpoints:**
- `POST /api/v1/submissions/{id}/upload` — upload a file (multipart/form-data)
- `GET /api/v1/submissions/{id}/files` — list all uploaded files with presigned URLs

### POST /submissions/{id}/upload
**Content-Type:** `multipart/form-data`  
**Form fields:**
- `file` — the file binary
- `type` — `"cover"` or `"attachment"`

**Validation:**
- Max file size: 50MB (checked before reading full body)
- Allowed MIME types for cover: `image/jpeg`, `image/png`, `image/webp`
- Allowed MIME types for attachments: `application/pdf`, `application/zip`, `video/mp4`, image types
- Only 1 cover image allowed per submission (updating replaces it)

**Success (201):**
```json
{
  "data": {
    "id": "uuid",
    "submissionId": "uuid",
    "url": "https://minio.example.com/uploads-covers/...",
    "contentType": "image/jpeg",
    "sizeBytes": 204800,
    "type": "cover",
    "uploadedAt": "..."
  },
  "meta": {...}
}
```

**Error cases:**
| Condition | HTTP | Code |
|-----------|------|------|
| File too large (> 50MB) | 413 | `FILE_TOO_LARGE` |
| Invalid MIME type | 400 | `INVALID_FILE_TYPE` |
| Past submission deadline (I11) | 422 | `DEADLINE_PASSED` |
| Not team member | 403 | `FORBIDDEN` |

### GET /submissions/{id}/files
**Response (200):**
```json
{
  "data": [
    { "id": "uuid", "url": "presigned-url", "contentType": "image/jpeg", "sizeBytes": 204800, "type": "cover" }
  ],
  "meta": {...}
}
```
Note: URLs are presigned (valid for 1 hour), generated on-demand from MinIO.

## Files to Create

### internal/shared/storage/minio.go — MinIO adapter (implements shared/port/storage.go)
```go
package storage

import (
    "context"
    "fmt"
    "io"
    "net/url"
    "time"

    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStorage struct {
    client *minio.Client
}

func NewMinIOStorage(endpoint, accessKey, secretKey string, useSSL bool) (*MinIOStorage, error) {
    client, err := minio.New(endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
        Secure: useSSL,
    })
    if err != nil {
        return nil, fmt.Errorf("create minio client: %w", err)
    }
    return &MinIOStorage{client: client}, nil
}

func (s *MinIOStorage) Upload(ctx context.Context, bucket, objectName, contentType string, size int64, reader io.Reader) (string, error) {
    _, err := s.client.PutObject(ctx, bucket, objectName, reader, size, minio.PutObjectOptions{
        ContentType: contentType,
    })
    if err != nil {
        return "", fmt.Errorf("upload to minio: %w", err)
    }
    return fmt.Sprintf("%s/%s/%s", s.client.EndpointURL(), bucket, objectName), nil
}

func (s *MinIOStorage) PresignedURL(ctx context.Context, bucket, objectName string) (string, error) {
    u, err := s.client.PresignedGetObject(ctx, bucket, objectName, time.Hour, url.Values{})
    if err != nil {
        return "", fmt.Errorf("presign url: %w", err)
    }
    return u.String(), nil
}

func (s *MinIOStorage) Delete(ctx context.Context, bucket, objectName string) error {
    return s.client.RemoveObject(ctx, bucket, objectName, minio.RemoveObjectOptions{})
}
```

### internal/submissions/port/out.go (add)
```go
type UploadRepository interface {
    Save(ctx context.Context, upload *domain.Upload) error
    FindBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]*domain.Upload, error)
    FindCoverBySubmissionID(ctx context.Context, submissionID uuid.UUID) (*domain.Upload, error)
    Delete(ctx context.Context, id uuid.UUID) error
}

// FileStorage is re-exported from shared/port — the submissions module uses this interface
type FileStorage = sharedport.FileStorage
```

### internal/submissions/domain/upload.go
```go
package domain

import (
    "errors"
    "strings"
    "github.com/google/uuid"
    "time"
)

var ErrFileTooLarge    = errors.New("file exceeds maximum size of 50MB")
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
```

### internal/submissions/usecase/upload.go
```go
func (s *UploadService) Upload(ctx context.Context, cmd port.UploadCommand) (*port.UploadDTO, error) {
    // 1. Load submission + verify team member
    // 2. Check deadline (I11) using sub.CheckCanEdit()
    // 3. Validate MIME type + size via domain.ValidateUpload()
    // 4. Build object name: "{submissionID}/{uploadType}/{uuid}.{ext}"
    // 5. Select bucket based on type: cover→"uploads-covers", attachment→"uploads-covers"
    // 6. If type == cover, delete old cover from MinIO + DB (replace)
    // 7. Upload to MinIO via storage port
    // 8. Record upload in uploads table
    // 9. If type == cover, update submissions.cover_url
    // 10. Return UploadDTO with URL
}
```

### Handler: multipart form parsing
```go
// Max 50MB + 10MB overhead for form fields
r.Body = http.MaxBytesReader(w, r.Body, (50+10)*1024*1024)
if err := r.ParseMultipartForm(10 << 20); err != nil {
    if strings.Contains(err.Error(), "too large") {
        response.Error(w, r, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "file exceeds 50MB limit")
        return
    }
    response.BadRequest(w, r, "VALIDATION_ERROR", "invalid multipart form")
    return
}
file, header, err := r.FormFile("file")
uploadType := r.FormValue("type")
```

## Tests Required

### Domain tests
- `TestValidateUpload_ValidCoverJPEG_ReturnsNil`
- `TestValidateUpload_AttachmentPDF_ReturnsNil`
- `TestValidateUpload_CoverPDF_ReturnsErrInvalidMIMEType`
- `TestValidateUpload_ExceedsMaxSize_ReturnsErrFileTooLarge`

### Use case tests (mock MinIO + mock upload repo)
- `TestUploadService_ValidCover_Uploads_RecordsMetadata`
- `TestUploadService_PastDeadline_Returns422` (I11)
- `TestUploadService_NotTeamMember_Returns403`
- `TestUploadService_ReplaceCover_DeletesOldFile`

### Handler tests (use small test file bytes)
- `TestUploadHandler_ValidJPEG_Returns201`
- `TestUploadHandler_TooLarge_Returns413`
- `TestUploadHandler_InvalidMIME_Returns400`

## Definition of Done
- [ ] `go test ./internal/submissions/...` → 100% green
- [ ] `POST /submissions/{id}/upload` with JPEG cover → 201 with URL
- [ ] File > 50MB → 413 `FILE_TOO_LARGE`
- [ ] Invalid MIME type → 400 `INVALID_FILE_TYPE`
- [ ] Past deadline → 422 `DEADLINE_PASSED`
- [ ] `GET /submissions/{id}/files` → presigned URLs (1-hour expiry)
- [ ] Cover replacement: uploading second cover deletes first from MinIO
- [ ] Upload metadata recorded in `uploads` table
