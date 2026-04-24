package resolvers

import (
	"bytes"
	"context"
	"testing"
)

func TestAuthorResolver_Resolve(t *testing.T) {
	r := AuthorResolver{}
	in := ResolveInput{
		TenantID:   "tenant-a",
		RevisionID: "rev-1",
		RevisionReader: fakeRevisionReader{
			author: AuthorInfo{
				UserID:      "u-1",
				DisplayName: "Jane Doe",
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

	author, ok := v1.Value.(AuthorInfo)
	if !ok {
		t.Fatalf("expected AuthorInfo value, got %T", v1.Value)
	}
	if author.UserID != "u-1" || author.DisplayName != "Jane Doe" {
		t.Fatalf("unexpected author: %#v", author)
	}
	if !bytes.Equal(v1.InputsHash, v2.InputsHash) {
		t.Fatal("expected stable hash across repeated resolves")
	}
}
