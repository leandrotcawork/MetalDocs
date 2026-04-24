package resolvers

import (
	"bytes"
	"context"
	"testing"
	"time"
)

type fakeWorkflowReader struct {
	approvers         []ApproverInfo
	finalApprovalDate time.Time
	err               error
}

func (f fakeWorkflowReader) GetApprovers(ctx context.Context, tenantID, revisionID string) ([]ApproverInfo, error) {
	return f.approvers, f.err
}

func (f fakeWorkflowReader) GetFinalApprovalDate(ctx context.Context, tenantID, revisionID string) (time.Time, error) {
	return f.finalApprovalDate, f.err
}

func TestApproversResolver_Resolve(t *testing.T) {
	r := ApproversResolver{}
	in := ResolveInput{
		TenantID:   "tenant-a",
		RevisionID: "rev-1",
		WorkflowReader: fakeWorkflowReader{
			approvers: []ApproverInfo{
				{
					UserID:      "u-1",
					DisplayName: "Jane",
					SignedAt:    time.Date(2026, time.April, 21, 14, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	v1, err := r.Resolve(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	v2, err := r.Resolve(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}

	approvers, ok := v1.Value.([]ApproverInfo)
	if !ok {
		t.Fatalf("expected []ApproverInfo value, got %T", v1.Value)
	}
	if len(approvers) != 1 || approvers[0].UserID != "u-1" {
		t.Fatalf("unexpected approvers: %#v", approvers)
	}
	if !bytes.Equal(v1.InputsHash, v2.InputsHash) {
		t.Fatal("expected stable hash across repeated resolves")
	}
}
