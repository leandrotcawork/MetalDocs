package resolvers

import (
	"context"
	"strings"
	"time"
)

type ApproversResolver struct{}

func (ApproversResolver) Key() string  { return "approvers" }
func (ApproversResolver) Version() int { return 1 }

func (ApproversResolver) Resolve(ctx context.Context, in ResolveInput) (ResolvedValue, error) {
	approvers, err := in.WorkflowReader.GetApprovers(ctx, in.TenantID, in.RevisionID)
	if err != nil {
		return ResolvedValue{}, err
	}

	var value string
	names := make([]string, 0, len(approvers))
	for _, a := range approvers {
		if a.DisplayName != "" {
			names = append(names, a.DisplayName)
		}
	}
	if len(names) == 0 {
		value = "[aguardando aprovação]"
	} else {
		value = strings.Join(names, ", ")
	}

	inputsHash, err := hashInputs(struct {
		TenantID   string `json:"tenant_id"`
		RevisionID string `json:"revision_id"`
	}{in.TenantID, in.RevisionID})
	if err != nil {
		return ResolvedValue{}, err
	}

	return ResolvedValue{
		Value:       value,
		ResolverKey: "approvers",
		ResolverVer: 1,
		InputsHash:  inputsHash,
		ComputedAt:  time.Now().UTC(),
	}, nil
}
