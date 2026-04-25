package application_test

// snapshot_wire_test.go — unit test asserting SnapshotService is called
// inside CreateDocument when wired via NewServiceWithSnapshot.
//
// Uses fakes already defined in service_test.go and service_cd_test.go:
// fakeRepo, fakeTplReader, noopAudit, fakeRegistryReader, fakeAuthzChecker,
// fakeProfileDefaultTemplateReader, strptr.

import (
	"context"
	"encoding/json"
	"testing"

	"metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/domain"
	registrydomain "metaldocs/internal/modules/registry/domain"
)

// wireSnapshotReader implements SnapshotTemplateReader for the wiring test.
type wireSnapshotReader struct{}

func (wireSnapshotReader) LoadForSnapshot(_ context.Context, _, _ string) (domain.TemplateSnapshot, error) {
	return domain.TemplateSnapshot{
		PlaceholderSchemaJSON: []byte(`{"placeholders":[]}`),
		CompositionJSON:       []byte(`{}`),
		BodyDocxBytes:         []byte("DOCX"),
		BodyDocxS3Key:         "s3://t/k",
	}, nil
}

// wireSnapshotWriter records WriteSnapshot calls.
type wireSnapshotWriter struct {
	called bool
	docID  string
}

func (w *wireSnapshotWriter) WriteSnapshot(_ context.Context, _, docID string, _ domain.TemplateSnapshot) error {
	w.called = true
	w.docID = docID
	return nil
}

func TestCreateDocument_SnapshotPopulated(t *testing.T) {
	const tenantID = "ffffffff-ffff-ffff-ffff-ffffffffffff"

	// fakeRepo is declared in service_test.go (always compiled).
	repo := &fakeRepo{createDocIDs: [3]string{"doc-snap-1", "rev-snap-1", "sess-snap-1"}}
	cd := &registrydomain.ControlledDocument{
		ID:              "cd-snap-1",
		TenantID:        tenantID,
		ProfileCode:     "PROC",
		ProcessAreaCode: "AREA-01",
		Status:          registrydomain.CDStatusActive,
	}

	writer := &wireSnapshotWriter{}
	snapSvc := application.NewSnapshotService(wireSnapshotReader{}, writer)

	svc := application.NewServiceWithSnapshot(
		repo,
		fakeDocgen{},
		&fakePresigner{hashReturn: "h_init"},
		fakeTplReader{},
		fakeFormVal{valid: true},
		&noopAudit{},
		&fakeRegistryReader{cd: cd},
		&fakeAuthzChecker{},
		&fakeProfileDefaultTemplateReader{id: strptr("tv-snap-1"), status: strptr("published")},
		snapSvc,
	)

	_, err := svc.CreateDocument(context.Background(), application.CreateDocumentInput{
		TenantID:             tenantID,
		ActorUserID:          "user-1",
		ControlledDocumentID: "cd-snap-1",
		TemplateVersionID:    "tv-snap-1",
		Name:                 "Test Doc",
		FormData:             json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	if !writer.called {
		t.Fatal("expected SnapshotService.SnapshotFromTemplate to be called, was not")
	}
	if writer.docID != "doc-snap-1" {
		t.Fatalf("snapshot written for wrong docID: got %q, want doc-snap-1", writer.docID)
	}
}
