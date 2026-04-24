package documentshttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v2dom "metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/iam/authz"
	"metaldocs/internal/modules/render/fanout"
)

type fakeReconstructService struct {
	entry fanout.ReconstructionEntry
	err   error
}

func (f fakeReconstructService) GetReconstruction(_ context.Context, _, _, _ string) (fanout.ReconstructionEntry, error) {
	if f.err != nil {
		return fanout.ReconstructionEntry{}, f.err
	}
	return f.entry, nil
}

func newReconstructReq(docID string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v2/documents/"+docID+"/reconstruct", nil)
	req.SetPathValue("id", docID)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req.Header.Set("X-User-ID", "user-1")
	return req
}

func TestReconstructHandler_SuccessReturnsEntry(t *testing.T) {
	ts := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	h := NewReconstructHandler(fakeReconstructService{entry: fanout.ReconstructionEntry{
		RenderedAt:       ts,
		EigenpalVer:      "eigenpal-1.2.3",
		DocxtemplaterVer: "docxtemplater-3.67.2",
		BytesHash:        "abc123",
		MatchesOriginal:  true,
	}})

	rec := httptest.NewRecorder()
	h.HandleReconstruct(rec, newReconstructReq("doc-1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var body fanout.ReconstructionEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.RenderedAt.UTC() != ts {
		t.Fatalf("rendered_at=%s want=%s", body.RenderedAt.UTC().Format(time.RFC3339), ts.Format(time.RFC3339))
	}
	if body.EigenpalVer != "eigenpal-1.2.3" || body.DocxtemplaterVer != "docxtemplater-3.67.2" || body.BytesHash != "abc123" || !body.MatchesOriginal {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestReconstructHandler_CapabilityDeniedReturns403(t *testing.T) {
	h := NewReconstructHandler(fakeReconstructService{err: authz.ErrCapabilityDenied{Capability: "doc.reconstruct", AreaCode: "AREA9", ActorID: "user-1"}})

	rec := httptest.NewRecorder()
	h.HandleReconstruct(rec, newReconstructReq("doc-1"))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestReconstructHandler_NotFoundReturns404(t *testing.T) {
	h := NewReconstructHandler(fakeReconstructService{err: v2dom.ErrNotFound})

	rec := httptest.NewRecorder()
	h.HandleReconstruct(rec, newReconstructReq("missing"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestReconstructHandler_InternalReturns500(t *testing.T) {
	h := NewReconstructHandler(fakeReconstructService{err: errors.New("boom")})

	rec := httptest.NewRecorder()
	h.HandleReconstruct(rec, newReconstructReq("doc-1"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
