package domain

import (
	"context"
	"database/sql"
)

type DBExecutor interface {
	QueryRowContext(ctx context.Context, sql string, args ...any) *sql.Row
	ExecContext(ctx context.Context, sql string, args ...any) (sql.Result, error)
}

type SequenceAllocator interface {
	NextAndIncrement(ctx context.Context, tx DBExecutor, tenantID, profileCode string) (int, error)
	EnsureCounter(ctx context.Context, tenantID, profileCode string) error
}
