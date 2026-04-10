package httpdelivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents/application"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

type recordingReleaseAuthChecker struct {
	allowed    bool
	userID     string
	documentID string
	calls      int
}

func (c *recordingReleaseAuthChecker) CanApprove(userID, documentID string) bool {
	c.calls++
	c.userID = userID
	c.documentID = documentID
	return c.allowed
}

func newTestReleaseHandler(t interface{ Helper() }) *ReleaseHandler {
	t.Helper()
	return NewReleaseHandler(nil)
}

func TestReleaseHandler_RequiresApprover(t *testing.T) {
	handler := newTestReleaseHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/documents/PO-118/release", nil)
	rec := httptest.NewRecorder()

	handler.Release(rec, req)

	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d or %d", rec.Code, http.StatusUnauthorized, http.StatusForbidden)
	}
}

func TestReleaseHandler_UnauthenticatedRequest(t *testing.T) {
	handler := newTestReleaseHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/documents/PO-118/release", nil)
	rec := httptest.NewRecorder()

	handler.Release(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestReleaseHandler_DeniedApprover(t *testing.T) {
	checker := &recordingReleaseAuthChecker{allowed: false}
	handler := NewReleaseHandler(checker)

	req := httptest.NewRequest(http.MethodPost, "/api/documents/PO-118/release", nil)
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "user-123", nil))
	rec := httptest.NewRecorder()

	handler.Release(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

type stubReleaseService struct {
	called bool
}

func (s *stubReleaseService) ReleaseDraft(_ context.Context, _ application.ReleaseInput) error {
	s.called = true
	return nil
}

func TestReleaseHandler_AllowedApprover(t *testing.T) {
	checker := &recordingReleaseAuthChecker{allowed: true}
	svc := &stubReleaseService{}
	handler := NewReleaseHandler(checker).WithReleaseService(svc)

	draftID := uuid.New().String()
	body := `{"draft_id":"` + draftID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/documents/PO-118/release", strings.NewReader(body))
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "user-123", nil))
	rec := httptest.NewRecorder()

	handler.Release(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if checker.calls != 1 {
		t.Fatalf("checker calls = %d, want %d", checker.calls, 1)
	}
	if checker.userID != "user-123" {
		t.Fatalf("checker userID = %q, want %q", checker.userID, "user-123")
	}
	if checker.documentID != "PO-118" {
		t.Fatalf("checker documentID = %q, want %q", checker.documentID, "PO-118")
	}
	if !svc.called {
		t.Fatal("expected release service to be called")
	}
}
