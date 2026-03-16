package noop

import (
	"context"

	"metaldocs/internal/platform/messaging"
)

type Publisher struct{}

func NewPublisher() *Publisher {
	return &Publisher{}
}

func (p *Publisher) Publish(_ context.Context, _ messaging.Event) error {
	return nil
}
