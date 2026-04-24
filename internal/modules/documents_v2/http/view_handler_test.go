package documentshttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	v2domain "metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/iam/authz"
)

type fakeViewService struct {
	result ViewResult
	err    error
}

func (f fakeViewService) GetViewURL(_ context.Context, _, _, _ string) (ViewResult, error) {
	if f.err != nil {
		return ViewResult{}, f.err
	}
	return f.result, nil
}

func newViewReq(docID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v2/documents/"+docID+"/view", nil)
	req.SetPathValue("id", docID)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-User-ID", "user-1")
	return req
}

func TestViewHandler_ApprovedReturnsSignedURL(t *testing.T) {
	h := NewViewHandler(fakeViewService{result: ViewResult{SignedURL: "https://s3.example/signed?x=1"}})

	rec := httptest.NewRecorder()
	h.HandleView(rec, newViewReq("doc-1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		SignedURL string `json:"signed_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.SignedURL != "https://s3.example/signed?x=1" {
		t.Errorf("signed_url = %q", body.SignedURL)
	}
}

func TestViewHandler_PublishedReturnsSignedURL(t *testing.T) {
	// Handler does not distinguish status — service does. From handler
	// perspective the behavior is identical to approved: success with URL.
	h := NewViewHandler(fakeViewService{result: ViewResult{SignedURL: "https://s3.example/pub"}})

	rec := httptest.NewRecorder()
	h.HandleView(rec, newViewReq("doc-pub"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestViewHandler_NotApprovedReturns404(t *testing.T) {
	h := NewViewHandler(fakeViewService{err: v2domain.ErrNotFound})

	rec := httptest.NewRecorder()
	h.HandleView(rec, newViewReq("doc-draft"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", rec.Code)
	}
}

func TestViewHandler_MissingAreaGrantReturns403(t *testing.T) {
	h := NewViewHandler(fakeViewService{err: authz.ErrCapabilityDenied{Capability: "doc.view_published", AreaCode: "AREA9", ActorID: "user-1"}})

	rec := httptest.NewRecorder()
	h.HandleView(rec, newViewReq("doc-1"))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", rec.Code)
	}
}

func TestViewHandler_PDFPendingReturns404WithCode(t *testing.T) {
	h := NewViewHandler(fakeViewService{err: ErrPDFPending})

	rec := httptest.NewRecorder()
	h.HandleView(rec, newViewReq("doc-pending"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", rec.Code)
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "pdf_pending" {
		t.Errorf("error = %q, want pdf_pending", body.Error)
	}
}
