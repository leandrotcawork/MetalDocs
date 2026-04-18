package objectstore

import (
	"context"
	"errors"
	"time"

	"github.com/minio/minio-go/v7"
)

type Config struct {
	MaxSizeBytes int64
	TTL          time.Duration
}

type PresignContext struct {
	MaxSizeBytes int64
	TTL          time.Duration
}

func NewPresignContext(cfg Config) (*PresignContext, error) {
	if cfg.MaxSizeBytes <= 0 {
		return nil, errors.New("max size must be > 0")
	}
	if cfg.TTL <= 0 {
		return nil, errors.New("ttl must be > 0")
	}
	return &PresignContext{MaxSizeBytes: cfg.MaxSizeBytes, TTL: cfg.TTL}, nil
}

// TemplatePresigner implements application.Presigner.
type TemplatePresigner struct {
	client       *minio.Client
	bucket       string
	ttl          time.Duration
	maxSizeBytes int64
}

func NewTemplatePresigner(client *minio.Client, bucket string, ttl time.Duration, maxSizeBytes int64) *TemplatePresigner {
	return &TemplatePresigner{client: client, bucket: bucket, ttl: ttl, maxSizeBytes: maxSizeBytes}
}

func (p *TemplatePresigner) PresignTemplateDocxPUT(ctx context.Context, tenantID, templateID string, versionNum int) (string, string, error) {
	key := TemplateDocxKey(tenantID, templateID, versionNum)
	u, err := p.client.PresignedPutObject(ctx, p.bucket, key, p.ttl)
	if err != nil {
		return "", "", err
	}
	return u.String(), key, nil
}

func (p *TemplatePresigner) PresignTemplateSchemaPUT(ctx context.Context, tenantID, templateID string, versionNum int) (string, string, error) {
	key := TemplateSchemaKey(tenantID, templateID, versionNum)
	u, err := p.client.PresignedPutObject(ctx, p.bucket, key, p.ttl)
	if err != nil {
		return "", "", err
	}
	return u.String(), key, nil
}

func (p *TemplatePresigner) PresignObjectGET(ctx context.Context, storageKey string) (string, error) {
	u, err := p.client.PresignedGetObject(ctx, p.bucket, storageKey, p.ttl, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
