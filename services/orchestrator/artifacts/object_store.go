package artifacts

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const maxObjectBytes = 10 << 20

// ErrObjectNotFound lets artifact adapters distinguish a missing blob from a
// transport or permission failure.
var ErrObjectNotFound = errors.New("object not found")

// ObjectStore is the cross-node blob boundary used by artifact adapters.
type ObjectStore interface {
	EnsureBucket(ctx context.Context) error
	Put(ctx context.Context, key string, data []byte, contentType string) error
	Get(ctx context.Context, key string) ([]byte, error)
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
	List(ctx context.Context, prefix string) ([]string, error)
	Delete(ctx context.Context, key string) error
}

// S3Config configures an S3-compatible endpoint (MinIO or cloud).
type S3Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	Secure    bool
}

// S3ObjectStore implements ObjectStore over MinIO/S3.
type S3ObjectStore struct {
	client *minio.Client
	bucket string
}

func NewS3ObjectStore(cfg S3Config) (*S3ObjectStore, error) {
	endpoint := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(cfg.Endpoint, "https://"), "http://"))
	bucket := strings.TrimSpace(cfg.Bucket)
	if endpoint == "" || bucket == "" || strings.Contains(bucket, "/") {
		return nil, fmt.Errorf("S3 endpoint and bucket are required")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("S3 credentials are required")
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.Secure,
		Region: strings.TrimSpace(cfg.Region),
	})
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}
	return &S3ObjectStore{client: client, bucket: bucket}, nil
}

func (s *S3ObjectStore) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
}

func (s *S3ObjectStore) Put(ctx context.Context, key string, data []byte, contentType string) error {
	if err := validateObjectKey(key); err != nil {
		return err
	}
	if len(data) > maxObjectBytes {
		return fmt.Errorf("object exceeds size limit")
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *S3ObjectStore) Get(ctx context.Context, key string) ([]byte, error) {
	if err := validateObjectKey(key); err != nil {
		return nil, err
	}
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer object.Close()
	data, err := io.ReadAll(io.LimitReader(object, maxObjectBytes+1))
	if err != nil {
		if isObjectNotFound(err) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}
	if len(data) > maxObjectBytes {
		return nil, fmt.Errorf("object exceeds size limit")
	}
	return data, nil
}

func (s *S3ObjectStore) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if err := validateObjectKey(key); err != nil {
		return "", err
	}
	if ttl <= 0 || ttl > 7*24*time.Hour {
		return "", fmt.Errorf("presign ttl must be between 0 and 7 days")
	}
	signed, err := s.client.PresignedGetObject(ctx, s.bucket, key, ttl, url.Values{})
	if err != nil {
		return "", err
	}
	return signed.String(), nil
}

func (s *S3ObjectStore) List(ctx context.Context, prefix string) ([]string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix != "" {
		if err := validateObjectKey(strings.TrimSuffix(prefix, "/")); err != nil {
			return nil, err
		}
	}
	objectCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true})
	keys := make([]string, 0)
	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}
		keys = append(keys, object.Key)
	}
	return keys, nil
}

func (s *S3ObjectStore) Delete(ctx context.Context, key string) error {
	if err := validateObjectKey(key); err != nil {
		return err
	}
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func isObjectNotFound(err error) bool {
	if errors.Is(err, ErrObjectNotFound) {
		return true
	}
	response := minio.ToErrorResponse(err)
	return response.Code == "NoSuchKey" || response.Code == "NoSuchBucket" ||
		response.StatusCode == http.StatusNotFound
}

func validateObjectKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" || strings.HasPrefix(key, "/") || strings.Contains(key, "..") || strings.ContainsAny(key, "\x00\\") {
		return fmt.Errorf("invalid object key")
	}
	return nil
}
