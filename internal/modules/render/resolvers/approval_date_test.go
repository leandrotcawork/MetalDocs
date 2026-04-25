package resolvers

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestApprovalDateResolver_Resolve(t *testing.T) {
	r := ApprovalDateResolver{}
	in := ResolveInput{
		TenantID:   "tenant-a",
		RevisionID: "rev-1",
		WorkflowReader: fakeWorkflowReader{
			finalApprovalDate: time.Date(2026, time.April, 21, 23, 15, 0, 0, time.FixedZone("UTC-3", -3*60*60)),
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

	if v1.Value != "2026-04-22" {
		t.Fatalf("expected approval date 2026-04-22, got %#v", v1.Value)
	}
	if !bytes.Equal(v1.InputsHash, v2.InputsHash) {
		t.Fatal("expected stable hash across repeated resolves")
	}
}
