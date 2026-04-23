package application

import (
	"errors"
	"strings"
	"testing"
)

func TestComputeContentHashFormat(t *testing.T) {
	got, err := ComputeContentHash(ContentHashInput{
		TenantID: "t-1", DocumentID: "doc-1", RevisionNumber: 1,
		FormData: map[string]any{"field": "value"},
	})
	if err != nil {
		t.Fatalf("ComputeContentHash: %v", err)
	}
	if len(got) != 64 {
		t.Errorf("hash length = %d; want 64", len(got))
	}
	if got != strings.ToLower(got) {
		t.Error("hash output should be lowercase")
	}
	// Deterministic.
	got2, _ := ComputeContentHash(ContentHashInput{
		TenantID: "t-1", DocumentID: "doc-1", RevisionNumber: 1,
		FormData: map[string]any{"field": "value"},
	})
	if got != got2 {
		t.Error("same input should produce same hash")
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

func TestComputeContentHashChangesWithValuesHash(t *testing.T) {
	base := ContentHashInput{
		TenantID:       "t",
		DocumentID:     "d",
		RevisionNumber: 1,
		FormData:       map[string]any{"k": "v"},
		ValuesHash:     "values-hash-a",
		SchemaHash:     "schema-hash-a",
	}
	h1, err := ComputeContentHash(base)
	if err != nil {
		t.Fatalf("base hash: %v", err)
	}

	changed := base
	changed.ValuesHash = "values-hash-b"
	h2, err := ComputeContentHash(changed)
	if err != nil {
		t.Fatalf("changed hash: %v", err)
	}

	if h1 == h2 {
		t.Fatal("expected different hash when values_hash changes")
	}
}
