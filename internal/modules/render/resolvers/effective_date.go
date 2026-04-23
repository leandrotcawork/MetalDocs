package resolvers

import (
	"context"
	"time"
)

type EffectiveDateResolver struct{}

func (EffectiveDateResolver) Key() string { return "effective_date" }

func (EffectiveDateResolver) Version() int { return 1 }

func (EffectiveDateResolver) Resolve(ctx context.Context, in ResolveInput) (ResolvedValue, error) {
	effectiveFrom, err := in.RevisionReader.GetEffectiveFrom(ctx, in.TenantID, in.RevisionID)
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
		Value:       effectiveFrom.UTC().Format("2006-01-02"),
		ResolverKey: "effective_date",
		ResolverVer: 1,
		InputsHash:  inputsHash,
		ComputedAt:  time.Now().UTC(),
	}, nil
}
