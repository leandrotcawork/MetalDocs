package messaging

import "context"

// Event is the stable envelope used by internal domain event publishers.
type Event struct {
	EventID        string
	EventType      string
	AggregateType  string
	AggregateID    string
	OccurredAtRFC3339 string
	Version        int
	IdempotencyKey string
	Producer       string
	TraceID        string
	Payload        map[string]any
}

// Publisher abstracts internal event delivery.
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}
