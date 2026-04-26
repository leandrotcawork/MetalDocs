package resolvers

import (
	"context"
	"testing"
)

type fakeDocReader struct{ title string }

func (f fakeDocReader) GetDocumentTitle(_ context.Context, _, _ string) (string, error) {
	return f.title, nil
}

func TestDocTitleResolver_Key(t *testing.T) {
	if got := (DocTitleResolver{}).Key(); got != "doc_title" {
		t.Fatalf("Key = %q, want %q", got, "doc_title")
	}
}

func TestDocTitleResolver_Resolve(t *testing.T) {
	r := DocTitleResolver{}
	in := ResolveInput{
		TenantID:       "t1",
		RevisionID:     "rev1",
		DocumentReader: fakeDocReader{title: "E2E Workflow Test - Rev 1"},
	}
	out, err := r.Resolve(context.Background(), in)
	if err != nil {
		t.Fatalf("Resolve err = %v", err)
	}
	if out.Value != "E2E Workflow Test - Rev 1" {
		t.Fatalf("Value = %q", out.Value)
	}
	if out.ResolverKey != "doc_title" {
		t.Fatalf("ResolverKey = %q", out.ResolverKey)
	}
}
