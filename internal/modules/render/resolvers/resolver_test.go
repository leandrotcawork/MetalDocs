package resolvers

import (
	"context"
	"testing"
	"time"
)

type fixedResolver struct{}

func (fixedResolver) Key() string { return "doc_code" }
func (fixedResolver) Version() int { return 1 }
func (fixedResolver) Resolve(ctx context.Context, in ResolveInput) (ResolvedValue, error) {
	return ResolvedValue{
		Value:       "QMS-0001",
		ResolverKey: "doc_code",
		ResolverVer: 1,
		InputsHash:  []byte("abc"),
		ComputedAt:  time.Unix(0, 0).UTC(),
	}, nil
}

func TestResolver_InterfaceShape(t *testing.T) {
	var r ComputedResolver = fixedResolver{}
	v, err := r.Resolve(context.Background(), ResolveInput{})
	if err != nil {
		t.Fatal(err)
	}
	if v.ResolverKey != "doc_code" {
		t.Fatalf("got %s", v.ResolverKey)
	}
}

func TestRegistry_Known_ReturnsAllResolvers(t *testing.T) {
	r := NewRegistry()
	r.Register(fixedResolver{})

	known := r.Known()
	if len(known) != 1 {
		t.Fatalf("expected 1 resolver, got %d", len(known))
	}
	if known["doc_code"] != 1 {
		t.Fatalf("expected doc_code version 1, got %d", known["doc_code"])
	}
}
