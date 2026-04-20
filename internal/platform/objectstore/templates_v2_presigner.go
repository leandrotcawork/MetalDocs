package objectstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"time"

	"github.com/minio/minio-go/v7"

	"metaldocs/internal/modules/templates_v2/domain"
)

// TemplatesV2Presigner implements templates_v2/application.Presigner.
type TemplatesV2Presigner struct {
	client       *minio.Client
	bucket       string
	maxSizeBytes int64
}

func NewTemplatesV2Presigner(client *minio.Client, bucket string, maxSizeBytes int64) *TemplatesV2Presigner {
	return &TemplatesV2Presigner{client: client, bucket: bucket, maxSizeBytes: maxSizeBytes}
}

func (p *TemplatesV2Presigner) PresignPUT(ctx context.Context, key string, expires time.Duration) (string, error) {
	u, err := p.client.PresignedPutObject(ctx, p.bucket, key, expires)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (p *TemplatesV2Presigner) HeadContentHash(ctx context.Context, key string) (string, error) {
	obj, err := p.client.GetObject(ctx, p.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		if isNoSuchKeyErr(err) {
			return "", domain.ErrUploadMissing
		}
		return "", err
	}
	defer obj.Close()

	if _, err := obj.Stat(); err != nil {
		if isNoSuchKeyErr(err) {
			return "", domain.ErrUploadMissing
		}
		return "", err
	}

	limit := p.maxSizeBytes
	if limit <= 0 {
		limit = 25 * 1024 * 1024
	}
	h := sha256.New()
	if _, err := io.Copy(h, io.LimitReader(obj, limit)); err != nil {
		if isNoSuchKeyErr(err) {
			return "", domain.ErrUploadMissing
		}
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (p *TemplatesV2Presigner) Delete(ctx context.Context, key string) error {
	err := p.client.RemoveObject(ctx, p.bucket, key, minio.RemoveObjectOptions{})
	if isNoSuchKeyErr(err) {
		return nil
	}
	return err
}
