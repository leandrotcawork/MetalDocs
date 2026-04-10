package mddm

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrImageNotFound = errors.New("image not found")

type ImageStorage interface {
	// Put stores image bytes idempotently. If an image with the same sha256 exists, returns the existing id.
	Put(ctx context.Context, sha256 string, mimeType string, bytes []byte) (uuid.UUID, error)
	// Get retrieves image bytes by id.
	Get(ctx context.Context, id uuid.UUID) (bytes []byte, mimeType string, err error)
	// Delete removes an image by id.
	Delete(ctx context.Context, id uuid.UUID) error
	// Exists checks if an image with this sha256 already exists.
	Exists(ctx context.Context, sha256 string) (id uuid.UUID, exists bool, err error)
}
