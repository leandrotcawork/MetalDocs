package application

import (
	"context"
	"errors"
	"testing"
	"time"

	v2dom "metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/documents_v2/repository"
	"metaldocs/internal/modules/render/resolvers"
	tmpldom "metaldocs/internal/modules/templates_v2/domain"
)

type fakeValuesReader struct {
	values []repository.PlaceholderValue
	err    error
}

func (f fakeValuesReader) ListValues(_ context.Context, _, _ string) ([]repository.PlaceholderValue, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.values, nil
}

type fakeFreezeFinalizer struct {
	calls int
	hash  []byte
	at    time.Time
	err   error
}

func (f *fakeFreezeFinalizer) WriteFreeze(_ context.Context, _, _ string, valuesHash []byte, frozenAt time.Time) error {
	if f.err != nil {
		return f.err
	}
	f.calls++
	f.hash = append([]byte(nil), valuesHash...)
	f.at = frozenAt
	return nil
}

type fakeResolverContextBuilder struct {
	input resolvers.ResolveInput
	err   error
	calls int
}

func (f *fakeResolverContextBuilder) Build(_ context.Context, _, _ string) (resolvers.ResolveInput, error) {
	if f.err != nil {
		return resolvers.ResolveInput{}, f.err
	}
	f.calls++
	return f.input, nil
}

type fixedResolver struct {
	key string
	ver int
	val any
}

func (r fixedResolver) Key() string { return r.key }
func (r fixedResolver) Version() int { return r.ver }
func (r fixedResolver) Resolve(_ context.Context, _ resolvers.ResolveInput) (resolvers.ResolvedValue, error) {
	return resolvers.ResolvedValue{
		Value:       r.val,
		ResolverKey: r.key,
		ResolverVer: r.ver,
		InputsHash:  []byte{0xaa},
		ComputedAt:  time.Now().UTC(),
	}, nil
}

func TestFreezeService_Freeze_ValidatesResolvesHashesAndFinalizes(t *testing.T) {
	resolverKey := "doc_code"
	schema := []tmpldom.Placeholder{
		{ID: "p_user", Required: true},
		{ID: "p_comp", Computed: true, ResolverKey: &resolverKey},
	}
	existing := []repository.PlaceholderValue{
		{PlaceholderID: "p_user", ValueText: strPtr("user-value"), Source: "user"},
	}
	writer := &fakeFillInWriter{}
	valuesRead := fakeValuesReader{values: existing}
	reg := resolvers.NewRegistry()
	reg.Register(fixedResolver{key: "doc_code", ver: 3, val: "DOC-001"})
	finalize := &fakeFreezeFinalizer{}
	ctxBuilder := &fakeResolverContextBuilder{input: resolvers.ResolveInput{TenantID: "t", RevisionID: "r"}}
	svc := NewFreezeService(fakeSchemaReader{placeholders: schema}, writer, valuesRead, reg, finalize, ctxBuilder)

	if err := svc.Freeze(context.Background(), "t", "r"); err != nil {
		t.Fatalf("Freeze error: %v", err)
	}
	if ctxBuilder.calls != 1 {
		t.Fatalf("expected context build call, got %d", ctxBuilder.calls)
	}
	if len(writer.upserts) != 1 {
		t.Fatalf("expected computed upsert, got %d", len(writer.upserts))
	}
	got := writer.upserts[0]
	if got.PlaceholderID != "p_comp" || got.Source != "computed" || got.ValueText == nil || *got.ValueText != "DOC-001" {
		t.Fatalf("bad computed upsert: %+v", got)
	}
	if got.ComputedFrom == nil || *got.ComputedFrom != "doc_code" || got.ResolverVersion == nil || *got.ResolverVersion != 3 {
		t.Fatalf("bad resolver metadata: %+v", got)
	}
	if finalize.calls != 1 {
		t.Fatalf("expected one finalize call, got %d", finalize.calls)
	}
	wantHash := v2dom.ComputeValuesHash(map[string]any{"p_user": "user-value", "p_comp": "DOC-001"})
	if bytesToHex(finalize.hash) != wantHash {
		t.Fatalf("hash mismatch: got %s want %s", bytesToHex(finalize.hash), wantHash)
	}
	if finalize.at.IsZero() {
		t.Fatal("expected frozenAt to be set")
	}
}

func TestFreezeService_Freeze_MissingRequiredUserPlaceholder(t *testing.T) {
	schema := []tmpldom.Placeholder{{ID: "p_user", Required: true}}
	svc := NewFreezeService(
		fakeSchemaReader{placeholders: schema},
		&fakeFillInWriter{},
		fakeValuesReader{},
		resolvers.NewRegistry(),
		&fakeFreezeFinalizer{},
		&fakeResolverContextBuilder{},
	)

	err := svc.Freeze(context.Background(), "t", "r")
	if !errors.Is(err, v2dom.ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}

func TestFreezeService_Freeze_ComputedMissingResolverKey(t *testing.T) {
	schema := []tmpldom.Placeholder{{ID: "p_comp", Computed: true}}
	svc := NewFreezeService(
		fakeSchemaReader{placeholders: schema},
		&fakeFillInWriter{},
		fakeValuesReader{},
		resolvers.NewRegistry(),
		&fakeFreezeFinalizer{},
		&fakeResolverContextBuilder{},
	)

	err := svc.Freeze(context.Background(), "t", "r")
	if !errors.Is(err, v2dom.ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}

func TestFreezeService_Freeze_UnknownResolverKey(t *testing.T) {
	resolverKey := "doc_code"
	schema := []tmpldom.Placeholder{{ID: "p_comp", Computed: true, ResolverKey: &resolverKey}}
	svc := NewFreezeService(
		fakeSchemaReader{placeholders: schema},
		&fakeFillInWriter{},
		fakeValuesReader{},
		resolvers.NewRegistry(),
		&fakeFreezeFinalizer{},
		&fakeResolverContextBuilder{},
	)

	err := svc.Freeze(context.Background(), "t", "r")
	if !errors.Is(err, tmpldom.ErrUnknownResolver) {
		t.Fatalf("expected ErrUnknownResolver, got %v", err)
	}
}

func strPtr(v string) *string { return &v }

func bytesToHex(b []byte) string {
	const hexDigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexDigits[v>>4]
		out[i*2+1] = hexDigits[v&0x0f]
	}
	return string(out)
}
