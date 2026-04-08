package httpdelivery

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

func newTestExportHandler(t testing.TB) *ExportHandler {
	t.Helper()
	return NewExportHandler()
}

func authExportRequest(t testing.TB, target string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	return req.WithContext(iamdomain.WithAuthContext(req.Context(), "user-123", nil))
}

func requireAPIError(t testing.TB, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d", rec.Code, wantStatus)
	}

	var envelope apiErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envelope.Error.Code != wantCode {
		t.Fatalf("code = %q, want %q", envelope.Error.Code, wantCode)
	}
}

func TestExportHandler_RequiresVersionID(t *testing.T) {
	handler := newTestExportHandler(t)

	req := authExportRequest(t, "/api/documents/PO-118/export/docx")
	rec := httptest.NewRecorder()

	handler.ExportDocx(rec, req)

	requireAPIError(t, rec, http.StatusNotFound, "VERSION_NOT_FOUND")
}

func TestExportHandler_UnauthenticatedRequest(t *testing.T) {
	handler := newTestExportHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/PO-118/export/docx?version_id="+uuid.NewString(), nil)
	rec := httptest.NewRecorder()

	handler.ExportDocx(rec, req)

	requireAPIError(t, rec, http.StatusUnauthorized, "AUTH_UNAUTHORIZED")
}

func TestExportHandler_RejectsInvalidMode(t *testing.T) {
	handler := newTestExportHandler(t)

	req := authExportRequest(t, "/api/documents/PO-118/export/docx?version_id="+uuid.NewString()+"&mode=invalid")
	rec := httptest.NewRecorder()

	handler.ExportDocx(rec, req)

	requireAPIError(t, rec, http.StatusBadRequest, "VALIDATION_ERROR")
}

func TestExportHandler_RejectsInvalidVersionID(t *testing.T) {
	handler := newTestExportHandler(t)

	req := authExportRequest(t, "/api/documents/PO-118/export/docx?version_id=not-a-uuid")
	rec := httptest.NewRecorder()

	handler.ExportDocx(rec, req)

	requireAPIError(t, rec, http.StatusBadRequest, "VALIDATION_ERROR")
}

func TestExportHandler_ReturnsDocxStubWhenVersionIDPresent(t *testing.T) {
	handler := newTestExportHandler(t)

	versionID := uuid.NewString()
	req := authExportRequest(t, "/api/documents/PO-118/export/docx?version_id="+versionID)
	rec := httptest.NewRecorder()

	handler.ExportDocx(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != docxContentType {
		t.Fatalf("content-type = %q, want %q", got, docxContentType)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "docx-stub" {
		t.Fatalf("body = %q, want %q", string(body), "docx-stub")
	}
}
