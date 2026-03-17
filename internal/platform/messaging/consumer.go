package messaging

import "context"

type Consumer interface {
	ClaimUnpublished(ctx context.Context, limit int) ([]Event, error)
	MarkPublished(ctx context.Context, eventIDs []string) error
	Release(ctx context.Context, eventIDs []string) error
}
