package application

import (
	"context"
	"errors"
	"testing"

	v2domain "metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/documents_v2/repository"
	templatesdomain "metaldocs/internal/modules/templates_v2/domain"
)

type fakeSchemaReader struct {
	placeholders []templatesdomain.Placeholder
	err          error
}

func (f fakeSchemaReader) LoadPlaceholderSchema(_ context.Context, _, _ string) ([]templatesdomain.Placeholder, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.placeholders, nil
}

type fakeFillInWriter struct {
	upserts []repository.PlaceholderValue
	err     error
}

func (f *fakeFillInWriter) UpsertValue(_ context.Context, v repository.PlaceholderValue) error {
	if f.err != nil {
		return f.err
	}
	f.upserts = append(f.upserts, v)
	return nil
}

func TestFillInService_ValueRejectedIfFailsRegex(t *testing.T) {
	re := "^[A-Z]{3}$"
	schema := []templatesdomain.Placeholder{{ID: "p1", Type: templatesdomain.PHText, Regex: &re}}
	svc := NewFillInServiceNoAuthz(fakeSchemaReader{placeholders: schema}, &fakeFillInWriter{})
	err := svc.SetPlaceholderValue(context.Background(), "tenant", "actor", "rev", "p1", "abc")
	if !errors.Is(err, v2domain.ErrValidationFailed) {
		t.Fatalf("got %v", err)
	}
}

func TestFillInService_ValueAcceptedIfMatches(t *testing.T) {
	re := "^[A-Z]{3}$"
	schema := []templatesdomain.Placeholder{{ID: "p1", Type: templatesdomain.PHText, Regex: &re}}
	writer := &fakeFillInWriter{}
	svc := NewFillInServiceNoAuthz(fakeSchemaReader{placeholders: schema}, writer)
	if err := svc.SetPlaceholderValue(context.Background(), "tenant", "actor", "rev", "p1", "ABC"); err != nil {
		t.Fatal(err)
	}
	if len(writer.upserts) != 1 || *writer.upserts[0].ValueText != "ABC" {
		t.Fatalf("bad upsert: %+v", writer.upserts)
	}
}

func TestFillInService_SetPlaceholderValue_ValidationMatrix(t *testing.T) {
	maxLen := 3
	minN := 10.0
	maxN := 20.0
	minD := "2026-04-01"
	maxD := "2026-04-30"

	schema := []templatesdomain.Placeholder{
		{ID: "req", Type: templatesdomain.PHText, Required: true},
		{ID: "len", Type: templatesdomain.PHText, MaxLength: &maxLen},
		{ID: "num", Type: templatesdomain.PHNumber, MinNumber: &minN, MaxNumber: &maxN},
		{ID: "date", Type: templatesdomain.PHDate, MinDate: &minD, MaxDate: &maxD},
		{ID: "sel", Type: templatesdomain.PHSelect, Options: []string{"A", "B"}},
	}

	cases := []struct {
		name          string
		placeholderID string
		value         string
		wantErr       bool
	}{
		{name: "required-empty", placeholderID: "req", value: "", wantErr: true},
		{name: "max-length", placeholderID: "len", value: "abcd", wantErr: true},
		{name: "number-too-low", placeholderID: "num", value: "9", wantErr: true},
		{name: "number-too-high", placeholderID: "num", value: "21", wantErr: true},
		{name: "number-ok", placeholderID: "num", value: "11"},
		{name: "date-before-min", placeholderID: "date", value: "2026-03-31", wantErr: true},
		{name: "date-after-max", placeholderID: "date", value: "2026-05-01", wantErr: true},
		{name: "date-ok", placeholderID: "date", value: "2026-04-20"},
		{name: "select-invalid", placeholderID: "sel", value: "X", wantErr: true},
		{name: "select-ok", placeholderID: "sel", value: "A"},
		{name: "unknown", placeholderID: "missing", value: "A", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writer := &fakeFillInWriter{}
			svc := NewFillInServiceNoAuthz(fakeSchemaReader{placeholders: schema}, writer)
			err := svc.SetPlaceholderValue(context.Background(), "tenant", "actor", "rev", tc.placeholderID, tc.value)
			if tc.wantErr {
				if !errors.Is(err, v2domain.ErrValidationFailed) {
					t.Fatalf("expected ErrValidationFailed, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(writer.upserts) != 1 {
				t.Fatalf("expected one upsert, got %d", len(writer.upserts))
			}
		})
	}
}

// --- Draft resolver wiring tests ---

type fakeDraftResolver struct {
	calls int
	err   error
}

func (f *fakeDraftResolver) ResolveComputedIfStale(_ context.Context, _, _ string) error {
	f.calls++
	return f.err
}

func TestFillInService_SetPlaceholderValue_TriggersDraftResolver(t *testing.T) {
	schema := []templatesdomain.Placeholder{{ID: "p1", Type: templatesdomain.PHText}}
	resolver := &fakeDraftResolver{}
	svc := NewFillInServiceNoAuthz(fakeSchemaReader{placeholders: schema}, &fakeFillInWriter{}).
		WithDraftResolver(resolver)

	if err := svc.SetPlaceholderValue(context.Background(), "t", "actor", "r", "p1", "hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolver.calls != 1 {
		t.Errorf("draft resolver calls=%d, want 1", resolver.calls)
	}
}

func TestFillInService_SetPlaceholderValue_DraftResolverError_IsBestEffort(t *testing.T) {
	schema := []templatesdomain.Placeholder{{ID: "p1", Type: templatesdomain.PHText}}
	resolver := &fakeDraftResolver{err: errors.New("resolver boom")}
	svc := NewFillInServiceNoAuthz(fakeSchemaReader{placeholders: schema}, &fakeFillInWriter{}).
		WithDraftResolver(resolver)

	// Resolver error must NOT propagate — best-effort.
	if err := svc.SetPlaceholderValue(context.Background(), "t", "actor", "r", "p1", "hello"); err != nil {
		t.Fatalf("expected nil but got: %v", err)
	}
}

// --- IAM user placeholder validation ---

type fakeIAMOptionsReader struct {
	opts []UserOption
	err  error
}

func (f *fakeIAMOptionsReader) ListUserOptions(_ context.Context, _ string) ([]UserOption, error) {
	return f.opts, f.err
}

func TestFillInService_UserPlaceholder_KnownUser_Accepted(t *testing.T) {
	schema := []templatesdomain.Placeholder{{ID: "p-user", Type: templatesdomain.PHUser}}
	iam := &fakeIAMOptionsReader{opts: []UserOption{
		{UserID: "u1", DisplayName: "Alice"},
	}}
	svc := NewFillInServiceNoAuthz(fakeSchemaReader{placeholders: schema}, &fakeFillInWriter{}).
		WithIAMReader(iam)

	if err := svc.SetPlaceholderValue(context.Background(), "t", "actor", "r", "p-user", "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFillInService_UserPlaceholder_UnknownUser_Returns422(t *testing.T) {
	schema := []templatesdomain.Placeholder{{ID: "p-user", Type: templatesdomain.PHUser}}
	iam := &fakeIAMOptionsReader{opts: []UserOption{
		{UserID: "u1", DisplayName: "Alice"},
	}}
	svc := NewFillInServiceNoAuthz(fakeSchemaReader{placeholders: schema}, &fakeFillInWriter{}).
		WithIAMReader(iam)

	err := svc.SetPlaceholderValue(context.Background(), "t", "actor", "r", "p-user", "unknown-uid")
	if !errors.Is(err, v2domain.ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}
