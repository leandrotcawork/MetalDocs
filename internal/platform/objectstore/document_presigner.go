package objectstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"

	"metaldocs/internal/modules/documents_v2/domain"
)

type DocumentPresigner struct {
	client       *minio.Client
	bucket       string
	ttl          time.Duration
	maxSizeBytes int64
}

func NewDocumentPresigner(client *minio.Client, bucket string, ttl time.Duration, maxSizeBytes int64) *DocumentPresigner {
	return &DocumentPresigner{
		client:       client,
		bucket:       bucket,
		ttl:          ttl,
		maxSizeBytes: maxSizeBytes,
	}
}

func (p *DocumentPresigner) PresignRevisionPUT(ctx context.Context, tenantID, docID, contentHash string) (string, string, error) {
	if p.client == nil {
		return "", "", errors.New("document presigner minio client is nil")
	}
	key := fmt.Sprintf("tenants/%s/documents/%s/revisions/%s.docx", tenantID, docID, contentHash)
	u, err := p.client.PresignedPutObject(ctx, p.bucket, key, p.ttl)
	if err != nil {
		return "", "", err
	}
	return u.String(), key, nil
}

func (p *DocumentPresigner) PresignObjectGET(ctx context.Context, storageKey string) (string, error) {
	if p.client == nil {
		return "", errors.New("document presigner minio client is nil")
	}
	u, err := p.client.PresignedGetObject(ctx, p.bucket, storageKey, p.ttl, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (p *DocumentPresigner) AdoptTempObject(ctx context.Context, tmpKey, finalKey string) error {
	if p.client == nil {
		return errors.New("document presigner minio client is nil")
	}
	src := minio.CopySrcOptions{
		Bucket: p.bucket,
		Object: tmpKey,
	}
	dst := minio.CopyDestOptions{
		Bucket: p.bucket,
		Object: finalKey,
	}
	if _, err := p.client.CopyObject(ctx, dst, src); err != nil {
		return err
	}
	if err := p.client.RemoveObject(ctx, p.bucket, tmpKey, minio.RemoveObjectOptions{}); err != nil && !isNoSuchKeyErr(err) {
		log.Printf("objectstore: adopt tmp cleanup failed for key=%s: %v", tmpKey, err)
	}
	return nil
}

func (p *DocumentPresigner) DeleteObject(ctx context.Context, key string) error {
	if p.client == nil {
		return errors.New("document presigner minio client is nil")
	}
	err := p.client.RemoveObject(ctx, p.bucket, key, minio.RemoveObjectOptions{})
	if isNoSuchKeyErr(err) {
		return nil
	}
	return err
}

func (p *DocumentPresigner) HashObject(ctx context.Context, key string) (string, error) {
	if p.client == nil {
		return "", errors.New("document presigner minio client is nil")
	}

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

	h := sha256.New()
	limit := p.maxSizeBytes
	if limit <= 0 {
		limit = 25 * 1024 * 1024
	}
	n, err := io.Copy(h, io.LimitReader(obj, limit+1))
	if err != nil {
		if isNoSuchKeyErr(err) {
			return "", domain.ErrUploadMissing
		}
		return "", err
	}
	if n > limit {
		return "", fmt.Errorf("object exceeds max size (%d bytes)", limit)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func isNoSuchKeyErr(err error) bool {
	if err == nil {
		return false
	}
	var resp minio.ErrorResponse
	if errors.As(err, &resp) && strings.EqualFold(resp.Code, "NoSuchKey") {
		return true
	}
	if strings.Contains(err.Error(), "NoSuchKey") {
		return true
	}
	var ue *url.Error
	if errors.As(err, &ue) && strings.Contains(ue.Error(), "NoSuchKey") {
		return true
	}
	return false
}
