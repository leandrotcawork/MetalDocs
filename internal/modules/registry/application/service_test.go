package application

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	registrydomain "metaldocs/internal/modules/registry/domain"
	taxonomydomain "metaldocs/internal/modules/taxonomy/domain"
)

func TestCreate_AutoCode(t *testing.T) {
	repo := newFakeControlledDocumentRepository()
	logger := &fakeGovernanceLogger{}
	seq := &fakeSequenceAllocator{next: 1}
	svc := NewRegistryService(nil, repo, seq, &fakeTemplateVersionChecker{}, &fakeProfileReader{}, &fakeAreaReader{}, logger)
	svc.now = func() time.Time { return time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC) }

	cd, err := svc.Create(context.Background(), CreateControlledDocumentCmd{
		TenantID:        "tenant-a",
		ProfileCode:     "po",
		ProcessAreaCode: "quality",
		Title:           "Welding Procedure",
		OwnerUserID:     "owner-1",
		ActorUserID:     "actor-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cd.Code != "PO-01" {
		t.Fatalf("expected PO-01, got %q", cd.Code)
	}
	if cd.SequenceNum == nil || *cd.SequenceNum != 1 {
		t.Fatalf("expected sequence 1, got %+v", cd.SequenceNum)
	}
	if len(logger.events) != 0 {
		t.Fatalf("expected zero governance events, got %+v", logger.events)
	}
}

func TestCreate_ManualCode(t *testing.T) {
	repo := newFakeControlledDocumentRepository()
	logger := &fakeGovernanceLogger{}
	svc := NewRegistryService(nil, repo, &fakeSequenceAllocator{next: 1}, &fakeTemplateVersionChecker{}, &fakeProfileReader{}, &fakeAreaReader{}, logger)

	cd, err := svc.Create(context.Background(), CreateControlledDocumentCmd{
		TenantID:         "tenant-a",
		ProfileCode:      "po",
		ProcessAreaCode:  "quality",
		Title:            "Legacy Document",
		OwnerUserID:      "owner-1",
		ActorUserID:      "actor-1",
		ManualCode:       stringPtr("PO-LEG-47"),
		ManualCodeReason: stringPtr("Legacy migration from spreadsheet"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cd.Code != "PO-LEG-47" {
		t.Fatalf("unexpected code: %q", cd.Code)
	}
	if cd.SequenceNum != nil {
		t.Fatalf("expected nil sequence for manual code, got %+v", cd.SequenceNum)
	}
	if len(logger.events) != 1 || logger.events[0].EventType != "numbering.override" {
		t.Fatalf("expected numbering.override event, got %+v", logger.events)
	}
}

func TestCreate_ManualCode_MissingReason(t *testing.T) {
	svc := NewRegistryService(nil, newFakeControlledDocumentRepository(), &fakeSequenceAllocator{next: 1}, &fakeTemplateVersionChecker{}, &fakeProfileReader{}, &fakeAreaReader{}, &fakeGovernanceLogger{})
	_, err := svc.Create(context.Background(), CreateControlledDocumentCmd{
		TenantID:        "tenant-a",
		ProfileCode:     "po",
		ProcessAreaCode: "quality",
		Title:           "Legacy Document",
		OwnerUserID:     "owner-1",
		ActorUserID:     "actor-1",
		ManualCode:      stringPtr("PO-LEG-47"),
	})
	if !errors.Is(err, registrydomain.ErrManualCodeReasonRequired) {
		t.Fatalf("expected ErrManualCodeReasonRequired, got %v", err)
	}
}

func TestCreate_ManualCode_ShortReason(t *testing.T) {
	svc := NewRegistryService(nil, newFakeControlledDocumentRepository(), &fakeSequenceAllocator{next: 1}, &fakeTemplateVersionChecker{}, &fakeProfileReader{}, &fakeAreaReader{}, &fakeGovernanceLogger{})
	_, err := svc.Create(context.Background(), CreateControlledDocumentCmd{
		TenantID:         "tenant-a",
		ProfileCode:      "po",
		ProcessAreaCode:  "quality",
		Title:            "Legacy Document",
		OwnerUserID:      "owner-1",
		ActorUserID:      "actor-1",
		ManualCode:       stringPtr("PO-LEG-47"),
		ManualCodeReason: stringPtr("too short"),
	})
	if !errors.Is(err, registrydomain.ErrManualCodeReasonRequired) {
		t.Fatalf("expected ErrManualCodeReasonRequired, got %v", err)
	}
}

func TestCreate_DuplicateCode(t *testing.T) {
	repo := newFakeControlledDocumentRepository()
	repo.codeExists = true
	svc := NewRegistryService(nil, repo, &fakeSequenceAllocator{next: 1}, &fakeTemplateVersionChecker{}, &fakeProfileReader{}, &fakeAreaReader{}, &fakeGovernanceLogger{})

	_, err := svc.Create(context.Background(), CreateControlledDocumentCmd{
		TenantID:         "tenant-a",
		ProfileCode:      "po",
		ProcessAreaCode:  "quality",
		Title:            "Welding Procedure",
		OwnerUserID:      "owner-1",
		ActorUserID:      "actor-1",
		ManualCode:       stringPtr("PO-01"),
		ManualCodeReason: stringPtr("Manual override due to migration"),
	})
	if !errors.Is(err, registrydomain.ErrCDCodeTaken) {
		t.Fatalf("expected ErrCDCodeTaken, got %v", err)
	}
}

func TestCreate_OverrideTemplate_GovernanceEvent(t *testing.T) {
	repo := newFakeControlledDocumentRepository()
	logger := &fakeGovernanceLogger{}
	checker := &fakeTemplateVersionChecker{byID: map[string]templateVersionState{
		"tpl-ovr-1": {status: stringPtr("published"), profileCode: "po"},
	}}
	svc := NewRegistryService(nil, repo, &fakeSequenceAllocator{next: 1}, checker, &fakeProfileReader{}, &fakeAreaReader{}, logger)

	_, err := svc.Create(context.Background(), CreateControlledDocumentCmd{
		TenantID:                  "tenant-a",
		ProfileCode:               "po",
		ProcessAreaCode:           "quality",
		Title:                     "Welding Procedure",
		OwnerUserID:               "owner-1",
		ActorUserID:               "actor-1",
		OverrideTemplateVersionID: stringPtr("tpl-ovr-1"),
		OverrideTemplateReason:    stringPtr("Emergency temporary override for legal form"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logger.events) != 1 || logger.events[0].EventType != "template.override" {
		t.Fatalf("expected template.override event, got %+v", logger.events)
	}
}

func TestCreate_OverrideTemplate_MissingReason(t *testing.T) {
	checker := &fakeTemplateVersionChecker{byID: map[string]templateVersionState{
		"tpl-ovr-1": {status: stringPtr("published"), profileCode: "po"},
	}}
	svc := NewRegistryService(nil, newFakeControlledDocumentRepository(), &fakeSequenceAllocator{next: 1}, checker, &fakeProfileReader{}, &fakeAreaReader{}, &fakeGovernanceLogger{})

	_, err := svc.Create(context.Background(), CreateControlledDocumentCmd{
		TenantID:                  "tenant-a",
		ProfileCode:               "po",
		ProcessAreaCode:           "quality",
		Title:                     "Welding Procedure",
		OwnerUserID:               "owner-1",
		ActorUserID:               "actor-1",
		OverrideTemplateVersionID: stringPtr("tpl-ovr-1"),
	})
	if !errors.Is(err, registrydomain.ErrOverrideReasonRequired) {
		t.Fatalf("expected ErrOverrideReasonRequired, got %v", err)
	}
}

func TestCreate_ProfileArchived(t *testing.T) {
	archivedAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	profiles := &fakeProfileReader{item: &taxonomydomain.DocumentProfile{Code: "po", TenantID: "tenant-a", ArchivedAt: &archivedAt}}
	svc := NewRegistryService(nil, newFakeControlledDocumentRepository(), &fakeSequenceAllocator{next: 1}, &fakeTemplateVersionChecker{}, profiles, &fakeAreaReader{}, &fakeGovernanceLogger{})

	_, err := svc.Create(context.Background(), CreateControlledDocumentCmd{
		TenantID:        "tenant-a",
		ProfileCode:     "po",
		ProcessAreaCode: "quality",
		Title:           "Welding Procedure",
		OwnerUserID:     "owner-1",
		ActorUserID:     "actor-1",
	})
	if !errors.Is(err, taxonomydomain.ErrProfileArchived) {
		t.Fatalf("expected ErrProfileArchived, got %v", err)
	}
}

func TestCreate_AreaArchived(t *testing.T) {
	archivedAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	areas := &fakeAreaReader{item: &taxonomydomain.ProcessArea{Code: "quality", TenantID: "tenant-a", ArchivedAt: &archivedAt}}
	svc := NewRegistryService(nil, newFakeControlledDocumentRepository(), &fakeSequenceAllocator{next: 1}, &fakeTemplateVersionChecker{}, &fakeProfileReader{}, areas, &fakeGovernanceLogger{})

	_, err := svc.Create(context.Background(), CreateControlledDocumentCmd{
		TenantID:        "tenant-a",
		ProfileCode:     "po",
		ProcessAreaCode: "quality",
		Title:           "Welding Procedure",
		OwnerUserID:     "owner-1",
		ActorUserID:     "actor-1",
	})
	if !errors.Is(err, taxonomydomain.ErrAreaArchived) {
		t.Fatalf("expected ErrAreaArchived, got %v", err)
	}
}

type fakeControlledDocumentRepository struct {
	codeExists bool
	created    *registrydomain.ControlledDocument
}

func newFakeControlledDocumentRepository() *fakeControlledDocumentRepository {
	return &fakeControlledDocumentRepository{}
}

func (f *fakeControlledDocumentRepository) GetByID(_ context.Context, _, _ string) (*registrydomain.ControlledDocument, error) {
	if f.created == nil {
		return nil, registrydomain.ErrCDNotFound
	}
	copy := *f.created
	return &copy, nil
}

func (f *fakeControlledDocumentRepository) GetByCode(_ context.Context, _, _, _ string) (*registrydomain.ControlledDocument, error) {
	return nil, registrydomain.ErrCDNotFound
}

func (f *fakeControlledDocumentRepository) CodeExists(_ context.Context, _, _, _ string) (bool, error) {
	return f.codeExists, nil
}

func (f *fakeControlledDocumentRepository) List(_ context.Context, _ string, _ registrydomain.CDFilter) ([]registrydomain.ControlledDocument, error) {
	return nil, nil
}

func (f *fakeControlledDocumentRepository) Create(_ context.Context, doc *registrydomain.ControlledDocument) error {
	copy := *doc
	f.created = &copy
	return nil
}

func (f *fakeControlledDocumentRepository) CreateTx(_ context.Context, _ *sql.Tx, doc *registrydomain.ControlledDocument) error {
	copy := *doc
	f.created = &copy
	return nil
}

func (f *fakeControlledDocumentRepository) UpdateStatus(_ context.Context, _, _ string, _ registrydomain.CDStatus, _ time.Time) error {
	return nil
}

type fakeSequenceAllocator struct {
	next int
}

func (f *fakeSequenceAllocator) NextAndIncrement(_ context.Context, _ registrydomain.DBExecutor, _, _ string) (int, error) {
	v := f.next
	f.next++
	return v, nil
}

func (f *fakeSequenceAllocator) EnsureCounter(_ context.Context, _, _ string) error {
	return nil
}

type templateVersionState struct {
	status      *string
	profileCode string
}

type fakeTemplateVersionChecker struct {
	byID map[string]templateVersionState
}

func (f *fakeTemplateVersionChecker) GetTemplateVersionState(_ context.Context, templateVersionID string) (*string, string, error) {
	if f.byID == nil {
		return nil, "", nil
	}
	item, ok := f.byID[templateVersionID]
	if !ok {
		return nil, "", nil
	}
	return item.status, item.profileCode, nil
}

type fakeProfileReader struct {
	item *taxonomydomain.DocumentProfile
}

func (f *fakeProfileReader) GetByCode(_ context.Context, tenantID, code string) (*taxonomydomain.DocumentProfile, error) {
	if f.item == nil {
		return &taxonomydomain.DocumentProfile{Code: code, TenantID: tenantID}, nil
	}
	copy := *f.item
	return &copy, nil
}

type fakeAreaReader struct {
	item *taxonomydomain.ProcessArea
}

func (f *fakeAreaReader) GetByCode(_ context.Context, tenantID, code string) (*taxonomydomain.ProcessArea, error) {
	if f.item == nil {
		return &taxonomydomain.ProcessArea{Code: code, TenantID: tenantID}, nil
	}
	copy := *f.item
	return &copy, nil
}

type fakeGovernanceLogger struct {
	events []taxonomydomain.GovernanceEvent
}

func (f *fakeGovernanceLogger) Log(_ context.Context, e taxonomydomain.GovernanceEvent) error {
	f.events = append(f.events, e)
	return nil
}

func stringPtr(v string) *string { return &v }
