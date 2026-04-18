package objectstore

import (
	"context"

	"github.com/minio/minio-go/v7"
)

func (p *DocumentPresigner) HeadObject(ctx context.Context, key string) (bool, error) {
	if p.client == nil {
		return false, nil
	}

	_, err := p.client.StatObject(ctx, p.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isNoSuchKeyErr(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (p *DocumentPresigner) SizeObject(ctx context.Context, key string) (int64, error) {
	if p.client == nil {
		return 0, nil
	}

	info, err := p.client.StatObject(ctx, p.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isNoSuchKeyErr(err) {
			return 0, nil
		}
		return 0, err
	}
	return info.Size, nil
}
