package domain

import "context"

type SequenceAllocator interface {
	NextAndIncrement(ctx context.Context, tx interface{}, tenantID, profileCode string) (int, error)
	EnsureCounter(ctx context.Context, tenantID, profileCode string) error
}
