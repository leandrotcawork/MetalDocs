package resolvers

import (
	"context"
	"time"
)

type AuthorResolver struct{}

func (AuthorResolver) Key() string { return "author" }

func (AuthorResolver) Version() int { return 1 }

func (AuthorResolver) Resolve(ctx context.Context, in ResolveInput) (ResolvedValue, error) {
	author, err := in.RevisionReader.GetAuthor(ctx, in.TenantID, in.RevisionID)
	if err != nil {
		return ResolvedValue{}, err
	}

	inputsHash, err := hashInputs(struct {
		TenantID   string `json:"tenant_id"`
		RevisionID string `json:"revision_id"`
	}{
		TenantID:   in.TenantID,
		RevisionID: in.RevisionID,
	})
	if err != nil {
		return ResolvedValue{}, err
	}

	return ResolvedValue{
		Value:       author,
		ResolverKey: "author",
		ResolverVer: 1,
		InputsHash:  inputsHash,
		ComputedAt:  time.Now().UTC(),
	}, nil
}
