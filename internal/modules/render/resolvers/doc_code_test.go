package resolvers

import (
	"bytes"
	"context"
	"testing"
)

type fakeRegistryReader struct {
	record ControlledDocumentInfo
	err    error
}

func (f fakeRegistryReader) GetControlledDocument(ctx context.Context, tenantID, controlledDocumentID string) (ControlledDocumentInfo, error) {
	return f.record, f.err
}

func TestDocCodeResolver_Resolve(t *testing.T) {
	r := DocCodeResolver{}
	in := ResolveInput{
		TenantID:             "tenant-a",
		ControlledDocumentID: "doc-1",
		RegistryReader: fakeRegistryReader{
			record: ControlledDocumentInfo{DocCode: "QMS-0001"},
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

	if v1.Value != "QMS-0001" {
		t.Fatalf("expected doc code QMS-0001, got %#v", v1.Value)
	}
	if !bytes.Equal(v1.InputsHash, v2.InputsHash) {
		t.Fatal("expected stable hash across repeated resolves")
	}
}
