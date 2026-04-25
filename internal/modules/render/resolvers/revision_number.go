package resolvers

import (
	"context"
	"time"
)

type RevisionNumberResolver struct{}

func (RevisionNumberResolver) Key() string { return "revision_number" }

func (RevisionNumberResolver) Version() int { return 1 }

func (RevisionNumberResolver) Resolve(ctx context.Context, in ResolveInput) (ResolvedValue, error) {
	revisionNumber, err := in.RevisionReader.GetRevisionNumber(ctx, in.TenantID, in.RevisionID)
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
		Value:       revisionNumber,
		ResolverKey: "revision_number",
		ResolverVer: 1,
		InputsHash:  inputsHash,
		ComputedAt:  time.Now().UTC(),
	}, nil
}
