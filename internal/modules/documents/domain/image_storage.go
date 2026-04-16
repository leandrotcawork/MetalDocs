package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrImageNotFound = errors.New("image not found")

type ImageStorage interface {
	Put(ctx context.Context, sha256 string, mimeType string, bytes []byte) (uuid.UUID, error)
	Get(ctx context.Context, id uuid.UUID) ([]byte, string, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
