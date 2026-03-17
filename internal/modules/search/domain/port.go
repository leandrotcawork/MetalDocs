package domain

import "context"

type Reader interface {
	ListDocuments(ctx context.Context) ([]Document, error)
	ListAccessPolicies(ctx context.Context, resourceScope, resourceID string) ([]AccessPolicy, error)
}
