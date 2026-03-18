package messaging

import (
	"context"
	"time"
)

type FailedEvent struct {
	EventID        string
	LastError      string
	NextAttemptAt  *time.Time
	DeadLetteredAt *time.Time
}

type Consumer interface {
	ClaimUnpublished(ctx context.Context, limit int) ([]Event, error)
	MarkPublished(ctx context.Context, eventIDs []string) error
	MarkFailed(ctx context.Context, failure FailedEvent) error
}
