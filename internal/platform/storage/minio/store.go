package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"metaldocs/internal/platform/config"
)

type Store struct {
	client           *miniogo.Client
	bucket           string
	autoCreateBucket bool
}

func NewStore(cfg config.AttachmentsConfig) (*Store, error) {
	client, err := miniogo.New(cfg.MinIOEndpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: cfg.MinIOUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &Store{
		client:           client,
		bucket:           cfg.MinIOBucket,
		autoCreateBucket: cfg.MinIOAutoCreateBucket,
	}, nil
}

func (s *Store) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check minio bucket: %w", err)
	}
	if exists {
		return nil
	}
	if !s.autoCreateBucket {
		return fmt.Errorf("minio bucket %q does not exist and auto create is disabled", s.bucket)
	}
	if err := s.client.MakeBucket(ctx, s.bucket, miniogo.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("create minio bucket: %w", err)
	}
	return nil
}

func (s *Store) Save(ctx context.Context, storageKey string, content []byte) error {
	reader := bytes.NewReader(content)
	_, err := s.client.PutObject(ctx, s.bucket, storageKey, reader, int64(len(content)), miniogo.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("save attachment content in minio: %w", err)
	}
	return nil
}

func (s *Store) Open(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, s.bucket, storageKey, miniogo.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("open attachment content from minio: %w", err)
	}
	if _, err := object.Stat(); err != nil {
		_ = object.Close()
		return nil, fmt.Errorf("stat attachment content from minio: %w", err)
	}
	return object, nil
}

func (s *Store) Delete(ctx context.Context, storageKey string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, storageKey, miniogo.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete attachment content from minio: %w", err)
	}
	return nil
}
