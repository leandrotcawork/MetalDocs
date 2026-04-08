package httpdelivery

import (
	"net/http"
	"net/http/httptest"
	"testing"

	iamdomain "metaldocs/internal/modules/iam/domain"
)

func TestReleaseHandler_RequiresApprover(t *testing.T) {
	handler := newTestReleaseHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/documents/PO-118/release", nil)
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "user-123", nil))
	rec := httptest.NewRecorder()

	handler.Release(rec, req)

	if rec.Code != http.StatusForbidden && rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d or %d", rec.Code, http.StatusUnauthorized, http.StatusForbidden)
	}
}
