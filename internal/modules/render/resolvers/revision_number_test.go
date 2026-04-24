package resolvers

import (
	"bytes"
	"context"
	"testing"
	"time"
)

type fakeRevisionReader struct {
	revisionNumber int64
	effectiveFrom  time.Time
	author         AuthorInfo
	err            error
}

func (f fakeRevisionReader) GetRevisionNumber(ctx context.Context, tenantID, revisionID string) (int64, error) {
	return f.revisionNumber, f.err
}

func (f fakeRevisionReader) GetEffectiveFrom(ctx context.Context, tenantID, revisionID string) (time.Time, error) {
	return f.effectiveFrom, f.err
}

func (f fakeRevisionReader) GetAuthor(ctx context.Context, tenantID, revisionID string) (AuthorInfo, error) {
	return f.author, f.err
}

func TestRevisionNumberResolver_Resolve(t *testing.T) {
	r := RevisionNumberResolver{}
	in := ResolveInput{
		TenantID:   "tenant-a",
		RevisionID: "rev-1",
		RevisionReader: fakeRevisionReader{
			revisionNumber: 7,
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

	if v1.Value != int64(7) {
		t.Fatalf("expected revision number 7, got %#v", v1.Value)
	}
	if !bytes.Equal(v1.InputsHash, v2.InputsHash) {
		t.Fatal("expected stable hash across repeated resolves")
	}
}
