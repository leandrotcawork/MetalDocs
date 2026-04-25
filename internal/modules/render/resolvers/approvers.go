package resolvers

import (
	"context"
	"time"
)

type ApproversResolver struct{}

func (ApproversResolver) Key() string { return "approvers" }

func (ApproversResolver) Version() int { return 1 }

func (ApproversResolver) Resolve(ctx context.Context, in ResolveInput) (ResolvedValue, error) {
	approvers, err := in.WorkflowReader.GetApprovers(ctx, in.TenantID, in.RevisionID)
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
		Value:       approvers,
		ResolverKey: "approvers",
		ResolverVer: 1,
		InputsHash:  inputsHash,
		ComputedAt:  time.Now().UTC(),
	}, nil
}
