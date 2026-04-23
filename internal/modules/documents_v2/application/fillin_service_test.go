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
	zones        []templatesdomain.EditableZone
	err          error
}

func (f fakeSchemaReader) LoadPlaceholderSchema(_ context.Context, _, _ string) ([]templatesdomain.Placeholder, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.placeholders, nil
}

func (f fakeSchemaReader) LoadZonesSchema(_ context.Context, _, _ string) ([]templatesdomain.EditableZone, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.zones, nil
}

type fakeFillInWriter struct {
	upserts []repository.PlaceholderValue
	zones   []repository.ZoneContent
	err     error
}

func (f *fakeFillInWriter) UpsertValue(_ context.Context, v repository.PlaceholderValue) error {
	if f.err != nil {
		return f.err
	}
	f.upserts = append(f.upserts, v)
	return nil
}

func (f *fakeFillInWriter) UpsertZoneContent(_ context.Context, z repository.ZoneContent) error {
	if f.err != nil {
		return f.err
	}
	f.zones = append(f.zones, z)
	return nil
}

func TestFillInService_ValueRejectedIfFailsRegex(t *testing.T) {
	re := "^[A-Z]{3}$"
	schema := []templatesdomain.Placeholder{{ID: "p1", Type: templatesdomain.PHText, Regex: &re}}
	svc := NewFillInService(fakeSchemaReader{placeholders: schema}, &fakeFillInWriter{})
	err := svc.SetPlaceholderValue(context.Background(), "tenant", "rev", "p1", "abc")
	if !errors.Is(err, v2domain.ErrValidationFailed) {
		t.Fatalf("got %v", err)
	}
}

func TestFillInService_ValueAcceptedIfMatches(t *testing.T) {
	re := "^[A-Z]{3}$"
	schema := []templatesdomain.Placeholder{{ID: "p1", Type: templatesdomain.PHText, Regex: &re}}
	writer := &fakeFillInWriter{}
	svc := NewFillInService(fakeSchemaReader{placeholders: schema}, writer)
	if err := svc.SetPlaceholderValue(context.Background(), "tenant", "rev", "p1", "ABC"); err != nil {
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
			svc := NewFillInService(fakeSchemaReader{placeholders: schema}, writer)
			err := svc.SetPlaceholderValue(context.Background(), "tenant", "rev", tc.placeholderID, tc.value)
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

func TestFillInService_SetZoneContent_RejectsDisallowedContent(t *testing.T) {
	maxLen := 50
	zones := []templatesdomain.EditableZone{
		{
			ID: "z1",
			ContentPolicy: templatesdomain.ContentPolicy{
				AllowTables:   false,
				AllowImages:   false,
				AllowHeadings: false,
				AllowLists:    false,
			},
			MaxLength: &maxLen,
		},
	}

	cases := []struct {
		name  string
		value string
	}{
		{name: "table", value: "<w:tbl><w:tr/></w:tbl>"},
		{name: "image", value: "<w:drawing/>"},
		{name: "heading", value: `<w:pStyle w:val="Heading1"/>`},
		{name: "list", value: "<w:numPr/>"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writer := &fakeFillInWriter{}
			svc := NewFillInService(fakeSchemaReader{zones: zones}, writer)
			err := svc.SetZoneContent(context.Background(), "tenant", "rev", "z1", tc.value)
			if !errors.Is(err, v2domain.ErrValidationFailed) {
				t.Fatalf("expected ErrValidationFailed, got %v", err)
			}
		})
	}
}

func TestFillInService_SetZoneContent_AcceptsAllowedContent(t *testing.T) {
	maxLen := 50
	zones := []templatesdomain.EditableZone{
		{
			ID: "z1",
			ContentPolicy: templatesdomain.ContentPolicy{
				AllowTables:   false,
				AllowImages:   false,
				AllowHeadings: false,
				AllowLists:    false,
			},
			MaxLength: &maxLen,
		},
	}

	writer := &fakeFillInWriter{}
	svc := NewFillInService(fakeSchemaReader{zones: zones}, writer)
	if err := svc.SetZoneContent(context.Background(), "tenant", "rev", "z1", "<w:p>ok</w:p>"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writer.zones) != 1 {
		t.Fatalf("expected 1 upsert, got %d", len(writer.zones))
	}
	if writer.zones[0].ContentOOXML != "<w:p>ok</w:p>" {
		t.Fatalf("bad content: %q", writer.zones[0].ContentOOXML)
	}
}

func TestFillInService_SetZoneContent_MaxLength(t *testing.T) {
	maxLen := 5
	zones := []templatesdomain.EditableZone{
		{
			ID: "z1",
			ContentPolicy: templatesdomain.ContentPolicy{
				AllowTables:   true,
				AllowImages:   true,
				AllowHeadings: true,
				AllowLists:    true,
			},
			MaxLength: &maxLen,
		},
	}

	svc := NewFillInService(fakeSchemaReader{zones: zones}, &fakeFillInWriter{})
	err := svc.SetZoneContent(context.Background(), "tenant", "rev", "z1", "<w:p>long</w:p>")
	if !errors.Is(err, v2domain.ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}
