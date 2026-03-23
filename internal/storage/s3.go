package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type ObjectInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
}

type S3Client interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	List(ctx context.Context, prefix string) ([]ObjectInfo, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

type Config struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Bucket    string
	Timeout   time.Duration
}

type s3Client struct {
	client   *minio.Client
	bucket   string
	timeout  time.Duration
	maxRetry int
}

func NewS3Client(cfg Config) (S3Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &s3Client{
		client:   client,
		bucket:   cfg.Bucket,
		timeout:  cfg.Timeout,
		maxRetry: 3,
	}, nil
}

func (s *s3Client) Upload(ctx context.Context, key string, reader io.Reader, size int64) error {
	var lastErr error
	for attempt := 0; attempt < s.maxRetry; attempt++ {
		uploadCtx, cancel := context.WithTimeout(ctx, s.timeout)
		_, err := s.client.PutObject(uploadCtx, s.bucket, key, reader, size, minio.PutObjectOptions{})
		cancel()

		if err == nil {
			return nil
		}
		lastErr = err
		
		// Exponential backoff
		if attempt < s.maxRetry-1 {
			time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
		}
	}
	return fmt.Errorf("upload failed after %d attempts: %w", s.maxRetry, lastErr)
}

func (s *s3Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	// Don't use timeout context here - reader needs to be valid after function returns
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	return obj, nil
}

func (s *s3Client) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	var result []ObjectInfo
	listCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	for obj := range s.client.ListObjects(listCtx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if obj.Err != nil {
			return nil, fmt.Errorf("list objects: %w", obj.Err)
		}
		result = append(result, ObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
		})
	}
	return result, nil
}

func (s *s3Client) Delete(ctx context.Context, key string) error {
	deleteCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	return s.client.RemoveObject(deleteCtx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *s3Client) Exists(ctx context.Context, key string) (bool, error) {
	statCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	_, err := s.client.StatObject(statCtx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}