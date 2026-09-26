package port

import (
	"context"
	"io"
)

type FileStorage interface {
	Upload(ctx context.Context, bucket, objectName, contentType string, size int64, reader io.Reader) (string, error)
	PresignedURL(ctx context.Context, bucket, objectName string) (string, error)
	Delete(ctx context.Context, bucket, objectName string) error
}
