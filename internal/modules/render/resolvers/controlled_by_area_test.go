package resolvers

import (
	"bytes"
	"context"
	"testing"
)

func TestControlledByAreaResolver_Resolve(t *testing.T) {
	r := ControlledByAreaResolver{}
	in := ResolveInput{
		TenantID:         "tenant-a",
		AreaCodeSnapshot: "ENG",
	}

	v1, err := r.Resolve(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	v2, err := r.Resolve(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}

	if v1.Value != "ENG" {
		t.Fatalf("expected area code ENG, got %#v", v1.Value)
	}
	if !bytes.Equal(v1.InputsHash, v2.InputsHash) {
		t.Fatal("expected stable hash across repeated resolves")
	}
}
