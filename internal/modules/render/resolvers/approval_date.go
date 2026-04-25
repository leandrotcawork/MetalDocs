package resolvers

import (
	"context"
	"time"
)

type ApprovalDateResolver struct{}

func (ApprovalDateResolver) Key() string { return "approval_date" }

func (ApprovalDateResolver) Version() int { return 1 }

func (ApprovalDateResolver) Resolve(ctx context.Context, in ResolveInput) (ResolvedValue, error) {
	approvalDate, err := in.WorkflowReader.GetFinalApprovalDate(ctx, in.TenantID, in.RevisionID)
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
		Value:       approvalDate.UTC().Format("2006-01-02"),
		ResolverKey: "approval_date",
		ResolverVer: 1,
		InputsHash:  inputsHash,
		ComputedAt:  time.Now().UTC(),
	}, nil
}
