package application

import (
	"context"
	"errors"
	"testing"

	"metaldocs/internal/modules/documents_v2/repository"
	"metaldocs/internal/modules/render/resolvers"
	tmpldom "metaldocs/internal/modules/templates_v2/domain"
)

// fakeComputedResolver is a resolver with configurable return value and call counter.
type fakeComputedResolver struct {
	key     string
	version int
	result  resolvers.ResolvedValue
	calls   int
}

func (f *fakeComputedResolver) Key() string    { return f.key }
func (f *fakeComputedResolver) Version() int   { return f.version }
func (f *fakeComputedResolver) Resolve(_ context.Context, _ resolvers.ResolveInput) (resolvers.ResolvedValue, error) {
	f.calls++
	return f.result, nil
}

func newDraftSvc(
	schema []tmpldom.Placeholder,
	existing []repository.PlaceholderValue,
	resolver *fakeComputedResolver,
) (*DraftResolverService, *fakeFillInWriter) {
	reg := resolvers.NewRegistry()
	reg.Register(resolver)

	writer := &fakeFillInWriter{}
	svc := NewDraftResolverService(
		fakeSchemaReader{placeholders: schema},
		writer,
		&fakeValuesReader{values: existing},
		reg,
		&fakeResolverContextBuilder{},
	)
	return svc, writer
}

func computedPH(id, key string) tmpldom.Placeholder {
	return tmpldom.Placeholder{ID: id, Computed: true, ResolverKey: &key}
}

func TestDraftResolver_FirstCall_WritesComputedValue(t *testing.T) {
	hash := []byte{0x01, 0x02}
	res := &fakeComputedResolver{
		key:     "doc_code",
		version: 1,
		result:  resolvers.ResolvedValue{ResolverKey: "doc_code", ResolverVer: 1, Value: "DC-001", InputsHash: hash},
	}
	svc, writer := newDraftSvc(
		[]tmpldom.Placeholder{computedPH("p1", "doc_code")},
		nil,
		res,
	)

	if err := svc.ResolveComputedIfStale(context.Background(), "t1", "r1"); err != nil {
		t.Fatalf("ResolveComputedIfStale: %v", err)
	}

	if res.calls != 1 {
		t.Errorf("resolver calls=%d, want 1", res.calls)
	}
	if len(writer.upserts) != 1 {
		t.Fatalf("upsert calls=%d, want 1", len(writer.upserts))
	}
	v := writer.upserts[0]
	if v.Source != "computed" {
		t.Errorf("Source=%q, want computed", v.Source)
	}
	if v.ComputedFrom == nil || *v.ComputedFrom != "doc_code" {
		t.Errorf("ComputedFrom=%v, want doc_code", v.ComputedFrom)
	}
	if string(v.InputsHash) != string(hash) {
		t.Errorf("InputsHash mismatch")
	}
}

func TestDraftResolver_SecondCall_SameHash_SkipsWrite(t *testing.T) {
	hash := []byte{0xAA, 0xBB}
	res := &fakeComputedResolver{
		key:     "doc_code",
		version: 1,
		result:  resolvers.ResolvedValue{ResolverKey: "doc_code", ResolverVer: 1, Value: "DC-001", InputsHash: hash},
	}
	existing := []repository.PlaceholderValue{
		{TenantID: "t1", RevisionID: "r1", PlaceholderID: "p1", InputsHash: hash},
	}
	svc, writer := newDraftSvc(
		[]tmpldom.Placeholder{computedPH("p1", "doc_code")},
		existing,
		res,
	)

	if err := svc.ResolveComputedIfStale(context.Background(), "t1", "r1"); err != nil {
		t.Fatalf("ResolveComputedIfStale: %v", err)
	}

	if res.calls != 1 {
		t.Errorf("resolver calls=%d, want 1", res.calls)
	}
	// Same hash → cache hit → no write.
	if len(writer.upserts) != 0 {
		t.Errorf("upsert calls=%d, want 0 (cache hit)", len(writer.upserts))
	}
}

func TestDraftResolver_DifferentHash_Rewrites(t *testing.T) {
	oldHash := []byte{0x01}
	newHash := []byte{0x02}
	res := &fakeComputedResolver{
		key:     "doc_code",
		version: 1,
		result:  resolvers.ResolvedValue{ResolverKey: "doc_code", ResolverVer: 1, Value: "DC-002", InputsHash: newHash},
	}
	existing := []repository.PlaceholderValue{
		{TenantID: "t1", RevisionID: "r1", PlaceholderID: "p1", InputsHash: oldHash},
	}
	svc, writer := newDraftSvc(
		[]tmpldom.Placeholder{computedPH("p1", "doc_code")},
		existing,
		res,
	)

	if err := svc.ResolveComputedIfStale(context.Background(), "t1", "r1"); err != nil {
		t.Fatalf("ResolveComputedIfStale: %v", err)
	}

	if res.calls != 1 {
		t.Errorf("resolver calls=%d, want 1", res.calls)
	}
	if len(writer.upserts) != 1 {
		t.Fatalf("upsert calls=%d, want 1 (hash changed)", len(writer.upserts))
	}
	if string(writer.upserts[0].InputsHash) != string(newHash) {
		t.Errorf("InputsHash not updated to new hash")
	}
}

func TestDraftResolver_UnknownResolverKey_ReturnsError(t *testing.T) {
	res := &fakeComputedResolver{key: "some_resolver", version: 1}
	svc, _ := newDraftSvc(
		[]tmpldom.Placeholder{computedPH("p1", "unknown_key")},
		nil,
		res,
	)

	err := svc.ResolveComputedIfStale(context.Background(), "t1", "r1")
	if err == nil {
		t.Fatal("expected error for unknown resolver key")
	}
	if !errors.Is(err, tmpldom.ErrUnknownResolver) {
		t.Errorf("err=%v, want wrapping ErrUnknownResolver", err)
	}
}
