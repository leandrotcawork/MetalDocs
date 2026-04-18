package application_test

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/platform/servicebus"
)

type fakeExportRepo struct {
	doc     *domain.Document
	rev     *domain.Revision
	exports map[string]*domain.Export
}

func (f *fakeExportRepo) GetDocument(_ context.Context, _, _ string) (*domain.Document, error) {
	if f.doc == nil {
		return nil, domain.ErrNotFound
	}
	return f.doc, nil
}

func (f *fakeExportRepo) GetRevision(_ context.Context, _, _ string) (*domain.Revision, error) {
	if f.rev == nil {
		return nil, domain.ErrNotFound
	}
	return f.rev, nil
}

func (f *fakeExportRepo) InsertExport(_ context.Context, e *domain.Export) (*domain.Export, error) {
	if f.exports == nil {
		f.exports = map[string]*domain.Export{}
	}
	k := hex.EncodeToString(e.CompositeHash)
	if found, ok := f.exports[k]; ok {
		return found, nil
	}
	inserted := *e
	inserted.ID = "exp_1"
	f.exports[k] = &inserted
	return &inserted, nil
}

func (f *fakeExportRepo) GetExportByHash(_ context.Context, _ string, compositeHash []byte) (*domain.Export, error) {
	if f.exports == nil {
		return nil, domain.ErrNotFound
	}
	k := hex.EncodeToString(compositeHash)
	exp, ok := f.exports[k]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return exp, nil
}

type fakeExportPresigner struct {
	headFound bool
	sizeBytes int64
}

func (f *fakeExportPresigner) PresignObjectGET(_ context.Context, storageKey string) (string, error) {
	return "https://example.local/" + storageKey, nil
}

func (f *fakeExportPresigner) HeadObject(_ context.Context, _ string) (bool, error) {
	return f.headFound, nil
}

func (f *fakeExportPresigner) SizeObject(_ context.Context, _ string) (int64, error) {
	return f.sizeBytes, nil
}

type fakePDFClient struct {
	calls int
	err   error
}

func (f *fakePDFClient) ConvertPDF(_ context.Context, _ servicebus.ConvertPDFRequest) (servicebus.ConvertPDFResult, error) {
	f.calls++
	if f.err != nil {
		return servicebus.ConvertPDFResult{}, f.err
	}
	return servicebus.ConvertPDFResult{OutputKey: "out.pdf", SizeBytes: 123, DocgenV2Version: "docgen-v2@test"}, nil
}

type fakeAudit struct {
	events []map[string]any
}

func (f *fakeAudit) Write(_ context.Context, tenantID, actorID, action, docID string, meta any) {
	f.events = append(f.events, map[string]any{
		"tenant_id": tenantID,
		"actor_id":  actorID,
		"action":    action,
		"doc_id":    docID,
		"meta":      meta,
	})
}

func newDoc(tenantID, documentID, revisionID string) *domain.Document {
	return &domain.Document{
		ID:                documentID,
		TenantID:          tenantID,
		TemplateVersionID: "tpl_ver_1",
		Name:              "Doc",
		CurrentRevisionID: revisionID,
	}
}

func newRev(documentID, revisionID string) *domain.Revision {
	return &domain.Revision{
		ID:         revisionID,
		DocumentID: documentID,
		StorageKey: "tenants/tenant_1/documents/doc_1/revisions/content.docx",
		ContentHash: hex.EncodeToString([]byte{
			0x01, 0x02, 0x03, 0x04,
		}),
	}
}

func newSvc(headFound bool, sizeBytes int64, docgenErr error) (*application.ExportService, *fakeExportRepo, *fakeExportPresigner, *fakePDFClient, *fakeAudit) {
	repo := &fakeExportRepo{
		doc:     newDoc("tenant_1", "doc_1", "rev_1"),
		rev:     newRev("doc_1", "rev_1"),
		exports: map[string]*domain.Export{},
	}
	presigner := &fakeExportPresigner{headFound: headFound, sizeBytes: sizeBytes}
	pdf := &fakePDFClient{err: docgenErr}
	audit := &fakeAudit{}
	svc := application.NewExportService(repo, presigner, pdf, audit, "docgen-v2@0.4.0", "grammar-v1")
	return svc, repo, presigner, pdf, audit
}

func cachedMeta(t *testing.T, evt map[string]any) bool {
	t.Helper()
	meta, ok := evt["meta"].(map[string]any)
	if !ok {
		t.Fatalf("event meta is not map: %#v", evt["meta"])
	}
	cached, ok := meta["cached"].(bool)
	if !ok {
		t.Fatalf("event cached flag is not bool: %#v", meta["cached"])
	}
	return cached
}

func TestExportPDF_ColdMiss_CallsDocgenAuditsCachedFalse(t *testing.T) {
	svc, _, _, pdf, audit := newSvc(false, 1024, nil)

	res, err := svc.ExportPDF(context.Background(), "tenant_1", "user_1", "doc_1", domain.RenderOptions{PaperSize: "A4", LandscapeP: false})
	if err != nil {
		t.Fatalf("ExportPDF() error = %v", err)
	}
	if res == nil || res.Export == nil {
		t.Fatalf("expected export result, got %#v", res)
	}
	if res.Cached {
		t.Fatalf("expected non-cached result")
	}
	if pdf.calls != 1 {
		t.Fatalf("expected one docgen call, got %d", pdf.calls)
	}
	if len(audit.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(audit.events))
	}
	if cachedMeta(t, audit.events[0]) {
		t.Fatalf("expected cached=false audit meta")
	}
}

func TestExportPDF_WarmHit_SkipsDocgenAuditsCachedTrue(t *testing.T) {
	svc, _, _, pdf, audit := newSvc(false, 1024, nil)

	_, err := svc.ExportPDF(context.Background(), "tenant_1", "user_1", "doc_1", domain.RenderOptions{PaperSize: "A4"})
	if err != nil {
		t.Fatalf("first ExportPDF() error = %v", err)
	}

	pdf.calls = 0
	audit.events = nil

	res, err := svc.ExportPDF(context.Background(), "tenant_1", "user_1", "doc_1", domain.RenderOptions{PaperSize: "A4"})
	if err != nil {
		t.Fatalf("second ExportPDF() error = %v", err)
	}
	if res == nil || !res.Cached {
		t.Fatalf("expected cached=true on second export, got %#v", res)
	}
	if pdf.calls != 0 {
		t.Fatalf("expected zero docgen calls on warm hit, got %d", pdf.calls)
	}
	if len(audit.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(audit.events))
	}
	if !cachedMeta(t, audit.events[0]) {
		t.Fatalf("expected cached=true audit meta")
	}
}

func TestExportPDF_GotenbergFailure_ReturnsDomainError_NoAudit(t *testing.T) {
	svc, _, _, _, audit := newSvc(false, 0, errors.New("gotenberg down"))

	_, err := svc.ExportPDF(context.Background(), "tenant_1", "user_1", "doc_1", domain.RenderOptions{PaperSize: "A4"})
	if !errors.Is(err, domain.ErrExportGotenbergFailed) {
		t.Fatalf("expected ErrExportGotenbergFailed, got %v", err)
	}
	if len(audit.events) != 0 {
		t.Fatalf("expected no audit events, got %d", len(audit.events))
	}
}

func TestExportPDF_S3HasObjectRowMissing_SkipsDocgen(t *testing.T) {
	svc, _, _, pdf, _ := newSvc(true, 2048, nil)

	res, err := svc.ExportPDF(context.Background(), "tenant_1", "user_1", "doc_1", domain.RenderOptions{PaperSize: "A4", LandscapeP: true})
	if err != nil {
		t.Fatalf("ExportPDF() error = %v", err)
	}
	if pdf.calls != 0 {
		t.Fatalf("expected zero docgen calls, got %d", pdf.calls)
	}
	if res == nil || res.Export == nil {
		t.Fatalf("expected export row, got %#v", res)
	}
	if res.Export.SizeBytes != 2048 {
		t.Fatalf("expected size_bytes=2048, got %d", res.Export.SizeBytes)
	}
}
