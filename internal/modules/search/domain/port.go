package domain

import "context"

type Reader interface {
	ListDocuments(ctx context.Context) ([]Document, error)
}
