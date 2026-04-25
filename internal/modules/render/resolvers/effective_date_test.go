package resolvers

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestEffectiveDateResolver_Resolve(t *testing.T) {
	r := EffectiveDateResolver{}
	in := ResolveInput{
		TenantID:   "tenant-a",
		RevisionID: "rev-1",
		RevisionReader: fakeRevisionReader{
			effectiveFrom: time.Date(2026, time.April, 2, 12, 30, 0, 0, time.FixedZone("UTC-3", -3*60*60)),
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

	if v1.Value != "2026-04-02" {
		t.Fatalf("expected effective date 2026-04-02, got %#v", v1.Value)
	}
	if !bytes.Equal(v1.InputsHash, v2.InputsHash) {
		t.Fatal("expected stable hash across repeated resolves")
	}
}
