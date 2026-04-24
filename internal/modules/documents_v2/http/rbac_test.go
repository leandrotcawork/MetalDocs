package documentshttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	v2domain "metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/iam/authz"
	"metaldocs/internal/modules/render/fanout"
)

// --- /view RBAC ---

func TestView_ReaderWithoutAreaGrant_Returns403(t *testing.T) {
	h := NewViewHandler(fakeViewService{
		err: authz.ErrCapabilityDenied{Capability: "doc.view_published", AreaCode: "AREA1", ActorID: "reader-1"},
	})
	rec := httptest.NewRecorder()
	h.HandleView(rec, newViewReq("doc-1"))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", rec.Code)
	}
}

func TestView_ReaderWithGrant_Returns200(t *testing.T) {
	h := NewViewHandler(fakeViewService{result: ViewResult{SignedURL: "https://s3.example/ok"}})
	rec := httptest.NewRecorder()
	h.HandleView(rec, newViewReq("doc-1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
}

// --- /reconstruct RBAC ---
// fakeReconstructService and newReconstructReq defined in reconstruct_handler_test.go

func TestReconstruct_QMSAdmin_Returns200(t *testing.T) {
	h := NewReconstructHandler(fakeReconstructService{
		entry: fanout.ReconstructionEntry{BytesHash: "aabb", MatchesOriginal: true},
	})
	rec := httptest.NewRecorder()
	h.HandleReconstruct(rec, newReconstructReq("doc-1"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
}

func TestReconstruct_MissingCapability_Returns403(t *testing.T) {
	h := NewReconstructHandler(fakeReconstructService{
		err: authz.ErrCapabilityDenied{Capability: "doc.reconstruct", AreaCode: "AREA1", ActorID: "author-1"},
	})
	rec := httptest.NewRecorder()
	h.HandleReconstruct(rec, newReconstructReq("doc-1"))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", rec.Code)
	}
}

func TestReconstruct_NotFound_Returns404(t *testing.T) {
	h := NewReconstructHandler(fakeReconstructService{err: v2domain.ErrNotFound})
	rec := httptest.NewRecorder()
	h.HandleReconstruct(rec, newReconstructReq("doc-missing"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", rec.Code)
	}
}
