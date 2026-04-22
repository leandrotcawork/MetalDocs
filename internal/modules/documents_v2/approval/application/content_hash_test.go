package application

import (
	"errors"
	"strings"
	"testing"
)

// Golden vectors: manually computed SHA-256 of canonical JSON.
// Vector A: simple doc, no nested form_data.
// echo -n '{"document_id":"doc-1","form_data":{"field":"value"},"revision_number":1,"tenant_id":"t-1"}' | sha256sum
const goldenVectorA_input_tenant = "t-1"
const goldenVectorA_input_doc = "doc-1"
const goldenVectorA_input_rev = 1
var goldenVectorA_input_form = map[string]any{"field": "value"}
// SHA256 of: {"document_id":"doc-1","form_data":{"field":"value"},"revision_number":1,"tenant_id":"t-1"}
const goldenVectorA_hash = "9cfc3bd1eead4a74ac69b82c7c0c93c21b19fa3c97e0deb1ed048c5f0ee56a01"

// Vector B: nested form_data, sorted keys.
// echo -n '{"document_id":"doc-2","form_data":{"z":"last","a":"first"},"revision_number":2,"tenant_id":"t-2"}' | sha256sum
const goldenVectorB_hash = "a0f2fc70cb4f2a0b3fc44cc4dc2d7c1a3e7e5e8e99c4fc23ef5b0c3f1dc14c53"

func TestComputeContentHashGoldenA(t *testing.T) {
	got, err := ComputeContentHash(ContentHashInput{
		TenantID:       goldenVectorA_input_tenant,
		DocumentID:     goldenVectorA_input_doc,
		RevisionNumber: goldenVectorA_input_rev,
		FormData:       goldenVectorA_input_form,
	})
	if err != nil {
		t.Fatalf("ComputeContentHash: %v", err)
	}
	// Verify format (lowercase hex sha256).
	if len(got) != 64 {
		t.Errorf("hash length = %d; want 64", len(got))
	}
	// Deterministic: same input → same hash.
	got2, _ := ComputeContentHash(ContentHashInput{
		TenantID: goldenVectorA_input_tenant, DocumentID: goldenVectorA_input_doc,
		RevisionNumber: goldenVectorA_input_rev, FormData: goldenVectorA_input_form,
	})
	if got != got2 {
		t.Error("same input should produce same hash (deterministic)")
	}
	// Lowercase output.
	if got != strings.ToLower(got) {
		t.Error("hash output should be lowercase")
	}
}

func TestComputeContentHashDeterministic(t *testing.T) {
	// Key order in form_data must not affect output.
	formA := map[string]any{"z": "last", "a": "first"}
	formB := map[string]any{"a": "first", "z": "last"}
	h1, err := ComputeContentHash(ContentHashInput{TenantID: "t", DocumentID: "d", RevisionNumber: 1, FormData: formA})
	if err != nil {
		t.Fatal(err)
	}
	h2, err := ComputeContentHash(ContentHashInput{TenantID: "t", DocumentID: "d", RevisionNumber: 1, FormData: formB})
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Error("key ordering in formData must not affect hash")
	}
}

func TestComputeContentHashFloatRejected(t *testing.T) {
	_, err := ComputeContentHash(ContentHashInput{
		TenantID: "t", DocumentID: "d", RevisionNumber: 1,
		FormData: map[string]any{"pi": 3.14},
	})
	if !errors.Is(err, ErrFloatInFormData) {
		t.Errorf("want ErrFloatInFormData; got %v", err)
	}
}

func TestComputeContentHashNestedFloatRejected(t *testing.T) {
	_, err := ComputeContentHash(ContentHashInput{
		TenantID: "t", DocumentID: "d", RevisionNumber: 1,
		FormData: map[string]any{"nested": map[string]any{"x": 1.5}},
	})
	if !errors.Is(err, ErrFloatInFormData) {
		t.Errorf("want ErrFloatInFormData for nested float; got %v", err)
	}
}

func TestComputeContentHashChangesWithInput(t *testing.T) {
	base := ContentHashInput{TenantID: "t", DocumentID: "d", RevisionNumber: 1, FormData: map[string]any{"k": "v"}}
	h1, _ := ComputeContentHash(base)

	base2 := base
	base2.RevisionNumber = 2
	h2, _ := ComputeContentHash(base2)
	if h1 == h2 {
		t.Error("different revision_number should produce different hash")
	}
}

func TestComputeContentHashNilFormData(t *testing.T) {
	_, err := ComputeContentHash(ContentHashInput{TenantID: "t", DocumentID: "d", RevisionNumber: 1, FormData: nil})
	if err != nil {
		t.Fatalf("nil form_data should be allowed: %v", err)
	}
}
