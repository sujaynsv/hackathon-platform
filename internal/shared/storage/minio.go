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
	
	scheme := "http"
	if s.client.EndpointURL().Scheme != "" {
		scheme = s.client.EndpointURL().Scheme
	} else if s.client.EndpointURL().Host != "" && s.client.EndpointURL().Port() == "443" {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s/%s/%s", scheme, s.client.EndpointURL().Host, bucket, objectName), nil
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
