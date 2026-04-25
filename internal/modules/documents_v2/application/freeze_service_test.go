package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	v2dom "metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/documents_v2/repository"
	"metaldocs/internal/modules/render/fanout"
	"metaldocs/internal/modules/render/resolvers"
	tmpldom "metaldocs/internal/modules/templates_v2/domain"
)

type fakeSnapshotReader struct {
	snap           v2dom.TemplateSnapshot
	valuesFrozenAt *time.Time
	err            error
}

func (f fakeSnapshotReader) ReadSnapshotWithFreezeAt(_ context.Context, _, _ string, _ ...repository.DBTX) (v2dom.TemplateSnapshot, *time.Time, error) {
	return f.snap, f.valuesFrozenAt, f.err
}

type fakeFinalDocxWriter struct {
	calls int
	key   string
	hash  []byte
	err   error
}

func (f *fakeFinalDocxWriter) WriteFinalDocx(_ context.Context, _, _, s3Key string, contentHash []byte, _ ...repository.DBTX) error {
	if f.err != nil {
		return f.err
	}
	f.calls++
	f.key = s3Key
	f.hash = append([]byte(nil), contentHash...)
	return nil
}

type fakeFanoutClient struct {
	req   fanout.FanoutRequest
	resp  fanout.FanoutResponse
	err   error
	calls int
}

func (f *fakeFanoutClient) Fanout(_ context.Context, req fanout.FanoutRequest) (fanout.FanoutResponse, error) {
	f.calls++
	f.req = req
	if f.err != nil {
		return fanout.FanoutResponse{}, f.err
	}
	return f.resp, nil
}

type fakeValuesReader struct {
	values []repository.PlaceholderValue
	err    error
	calls  int
}

func (f *fakeValuesReader) ListValues(_ context.Context, _, _ string) ([]repository.PlaceholderValue, error) {
	f.calls++
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

func (f *fakeFreezeFinalizer) WriteFreeze(_ context.Context, _, _ string, valuesHash []byte, frozenAt time.Time, _ ...repository.DBTX) error {
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

func (f *fakeResolverContextBuilder) Build(_ context.Context, _, _ string, _ ApproverContext) (resolvers.ResolveInput, error) {
	if f.err != nil {
		return resolvers.ResolveInput{}, f.err
	}
	f.calls++
	return f.input, nil
}

func (f *fakeResolverContextBuilder) BuildForDraft(_ context.Context, _, _ string) (resolvers.ResolveInput, error) {
	if f.err != nil {
		return resolvers.ResolveInput{}, f.err
	}
	return f.input, nil
}

type fixedResolver struct {
	key string
	ver int
	val any
}

func (r fixedResolver) Key() string  { return r.key }
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
	valuesRead := &fakeValuesReader{values: existing}
	reg := resolvers.NewRegistry()
	reg.Register(fixedResolver{key: "doc_code", ver: 3, val: "DOC-001"})
	finalize := &fakeFreezeFinalizer{}
	ctxBuilder := &fakeResolverContextBuilder{input: resolvers.ResolveInput{TenantID: "t", RevisionID: "r"}}
	snapReader := fakeSnapshotReader{snap: v2dom.TemplateSnapshot{
		BodyDocxS3Key:   "templates/body.docx",
		CompositionJSON: []byte(`{"header_sub_blocks":["h1"]}`),
	}}
	finalDocx := &fakeFinalDocxWriter{}
	fanoutClient := &fakeFanoutClient{resp: fanout.FanoutResponse{
		ContentHash:    "deadbeef00000000000000000000000000000000000000000000000000000000",
		FinalDocxS3Key: "final/r.docx",
		UnreplacedVars: []string{},
	}}
	svc := NewFreezeService(fakeSchemaReader{placeholders: schema}, writer, valuesRead, reg, finalize, ctxBuilder, snapReader, finalDocx, fanoutClient)

	if err := svc.Freeze(context.Background(), nil, "t", "r", ApproverContext{}); err != nil {
		t.Fatalf("Freeze error: %v", err)
	}
	if fanoutClient.calls != 1 {
		t.Fatalf("expected 1 fanout call, got %d", fanoutClient.calls)
	}
	if fanoutClient.req.BodyDocxS3Key != "templates/body.docx" {
		t.Errorf("fanout body key = %q", fanoutClient.req.BodyDocxS3Key)
	}
	if fanoutClient.req.PlaceholderValues["p_user"] != "user-value" || fanoutClient.req.PlaceholderValues["p_comp"] != "DOC-001" {
		t.Errorf("fanout placeholder values = %+v", fanoutClient.req.PlaceholderValues)
	}
	if string(fanoutClient.req.Composition) != `{"header_sub_blocks":["h1"]}` {
		t.Errorf("fanout composition = %s", fanoutClient.req.Composition)
	}
	if finalDocx.calls != 1 || finalDocx.key != "final/r.docx" {
		t.Fatalf("WriteFinalDocx calls=%d key=%q", finalDocx.calls, finalDocx.key)
	}
	if bytesToHex(finalDocx.hash) != "deadbeef00000000000000000000000000000000000000000000000000000000" {
		t.Errorf("content_hash = %s", bytesToHex(finalDocx.hash))
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
		&fakeValuesReader{},
		resolvers.NewRegistry(),
		&fakeFreezeFinalizer{},
		&fakeResolverContextBuilder{},
		fakeSnapshotReader{},
		&fakeFinalDocxWriter{},
		&fakeFanoutClient{},
	)

	err := svc.Freeze(context.Background(), nil, "t", "r", ApproverContext{})
	if !errors.Is(err, v2dom.ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}

func TestFreezeService_Freeze_ComputedMissingResolverKey(t *testing.T) {
	schema := []tmpldom.Placeholder{{ID: "p_comp", Computed: true}}
	svc := NewFreezeService(
		fakeSchemaReader{placeholders: schema},
		&fakeFillInWriter{},
		&fakeValuesReader{},
		resolvers.NewRegistry(),
		&fakeFreezeFinalizer{},
		&fakeResolverContextBuilder{},
		fakeSnapshotReader{},
		&fakeFinalDocxWriter{},
		&fakeFanoutClient{},
	)

	err := svc.Freeze(context.Background(), nil, "t", "r", ApproverContext{})
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
		&fakeValuesReader{},
		resolvers.NewRegistry(),
		&fakeFreezeFinalizer{},
		&fakeResolverContextBuilder{},
		fakeSnapshotReader{},
		&fakeFinalDocxWriter{},
		&fakeFanoutClient{},
	)

	err := svc.Freeze(context.Background(), nil, "t", "r", ApproverContext{})
	if !errors.Is(err, tmpldom.ErrUnknownResolver) {
		t.Fatalf("expected ErrUnknownResolver, got %v", err)
	}
}

func TestFreezeService_Freeze_FanoutErrorSkipsFinalDocxWrite(t *testing.T) {
	schema := []tmpldom.Placeholder{{ID: "p_user", Required: true}}
	existing := []repository.PlaceholderValue{
		{PlaceholderID: "p_user", ValueText: strPtr("value"), Source: "user"},
	}
	finalDocx := &fakeFinalDocxWriter{}
	fanoutClient := &fakeFanoutClient{err: errors.New("docgen down")}
	svc := NewFreezeService(
		fakeSchemaReader{placeholders: schema},
		&fakeFillInWriter{},
		&fakeValuesReader{values: existing},
		resolvers.NewRegistry(),
		&fakeFreezeFinalizer{},
		&fakeResolverContextBuilder{},
		fakeSnapshotReader{snap: v2dom.TemplateSnapshot{BodyDocxS3Key: "body", CompositionJSON: []byte(`{}`)}},
		finalDocx,
		fanoutClient,
	)

	err := svc.Freeze(context.Background(), nil, "t", "r", ApproverContext{})
	if err == nil || !containsStr(err.Error(), "fanout") {
		t.Fatalf("expected fanout error, got %v", err)
	}
	if finalDocx.calls != 0 {
		t.Fatalf("WriteFinalDocx should not run on fanout error, got %d calls", finalDocx.calls)
	}
}

func TestFreezeService_Freeze_DefaultsEmptyComposition(t *testing.T) {
	schema := []tmpldom.Placeholder{{ID: "p_user", Required: true}}
	existing := []repository.PlaceholderValue{
		{PlaceholderID: "p_user", ValueText: strPtr("v"), Source: "user"},
	}
	fanoutClient := &fakeFanoutClient{resp: fanout.FanoutResponse{
		ContentHash:    "aa" + "00000000000000000000000000000000000000000000000000000000000000",
		FinalDocxS3Key: "out.docx",
	}}
	svc := NewFreezeService(
		fakeSchemaReader{placeholders: schema},
		&fakeFillInWriter{},
		&fakeValuesReader{values: existing},
		resolvers.NewRegistry(),
		&fakeFreezeFinalizer{},
		&fakeResolverContextBuilder{},
		fakeSnapshotReader{snap: v2dom.TemplateSnapshot{BodyDocxS3Key: "body"}},
		&fakeFinalDocxWriter{},
		fanoutClient,
	)

	if err := svc.Freeze(context.Background(), nil, "t", "r", ApproverContext{}); err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	if string(fanoutClient.req.Composition) != `{}` {
		t.Fatalf("empty composition should default to {}, got %s", fanoutClient.req.Composition)
	}
	var raw json.RawMessage = fanoutClient.req.Composition
	_ = raw
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
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
