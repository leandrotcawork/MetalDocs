package resolvers

import (
	"context"
	"time"
)

type ControlledByAreaResolver struct{}

func (ControlledByAreaResolver) Key() string { return "controlled_by_area" }

func (ControlledByAreaResolver) Version() int { return 1 }

func (ControlledByAreaResolver) Resolve(ctx context.Context, in ResolveInput) (ResolvedValue, error) {
	inputsHash, err := hashInputs(struct {
		TenantID         string `json:"tenant_id"`
		AreaCodeSnapshot string `json:"area_code_snapshot"`
	}{
		TenantID:         in.TenantID,
		AreaCodeSnapshot: in.AreaCodeSnapshot,
	})
	if err != nil {
		return ResolvedValue{}, err
	}

	return ResolvedValue{
		Value:       in.AreaCodeSnapshot,
		ResolverKey: "controlled_by_area",
		ResolverVer: 1,
		InputsHash:  inputsHash,
		ComputedAt:  time.Now().UTC(),
	}, nil
}
