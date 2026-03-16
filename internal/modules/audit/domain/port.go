package domain

import (
	"context"
	"time"
)

type Event struct {
	ID           string
	OccurredAt   time.Time
	ActorID      string
	Action       string
	ResourceType string
	ResourceID   string
	PayloadJSON  string
	TraceID      string
}

type Writer interface {
	Record(ctx context.Context, event Event) error
}
