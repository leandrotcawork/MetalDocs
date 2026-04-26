package approvalhttp

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/http/contracts"
	"metaldocs/internal/modules/documents_v2/approval/repository"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

type fakeReadServiceGetInstance struct {
	instance *domain.Instance
	err      error

	gotTenantID   string
	gotActorID    string
	gotInstanceID string
}

func (f *fakeReadServiceGetInstance) LoadInstance(_ context.Context, _ *sql.DB, tenantID, actorID, instanceID string) (*domain.Instance, error) {
	f.gotTenantID = tenantID
	f.gotActorID = actorID
	f.gotInstanceID = instanceID
	if f.err != nil {
		return nil, f.err
	}
	return f.instance, nil
}

func (f *fakeReadServiceGetInstance) LoadActiveInstanceByDocument(_ context.Context, _ *sql.DB, _, _ string) (*domain.Instance, error) {
	return nil, nil
}

func (f *fakeReadServiceGetInstance) ListPendingForActor(_ context.Context, _ *sql.DB, _, _, _ string, _, _ int) ([]domain.Instance, error) {
	return nil, nil
}

func getInstanceTestMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v2/approval/instances/{instance_id}", h.GetInstanceHandler)
	return mux
}

func TestGetInstanceHandler_HappyPath(t *testing.T) {
	ts := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	fakeSvc := &fakeReadServiceGetInstance{
		instance: &domain.Instance{
			ID:          "inst-1",
			DocumentID:  "doc-1",
			TenantID:    "tenant-1",
			Status:      domain.InstanceInProgress,
			SubmittedBy: "actor-123",
			SubmittedAt: ts,
		},
	}

	h := &Handler{readSvc: fakeSvc}
	mux := getInstanceTestMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/approval/instances/inst-1", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "actor-1", []iamdomain.Role{}))
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("ETag"); got != "\"v1\"" {
		t.Fatalf("etag header = %q, want %q", got, "\"v1\"")
	}

	var out contracts.InstanceResponse
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.ID != "inst-1" {
		t.Fatalf("id = %q, want %q", out.ID, "inst-1")
	}
	if out.SubmittedAt != ts.Format(time.RFC3339) {
		t.Fatalf("submitted_at = %q, want %q", out.SubmittedAt, ts.Format(time.RFC3339))
	}
	if out.ETag != "\"v1\"" {
		t.Fatalf("etag body = %q, want %q", out.ETag, "\"v1\"")
	}
	if fakeSvc.gotTenantID != "tenant-1" || fakeSvc.gotActorID != "actor-1" || fakeSvc.gotInstanceID != "inst-1" {
		t.Fatalf("unexpected service args tenant=%q actor=%q instance=%q", fakeSvc.gotTenantID, fakeSvc.gotActorID, fakeSvc.gotInstanceID)
	}
}

func TestGetInstanceHandler_NotFound(t *testing.T) {
	fakeSvc := &fakeReadServiceGetInstance{err: repository.ErrNoActiveInstance}
	h := &Handler{readSvc: fakeSvc}
	mux := getInstanceTestMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/approval/instances/inst-missing", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "actor-1", []iamdomain.Role{}))
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestGetInstanceHandler_NoTenantHeader(t *testing.T) {
	fakeSvc := &fakeReadServiceGetInstance{
		instance: &domain.Instance{
			ID:          "inst-2",
			DocumentID:  "doc-2",
			TenantID:    "",
			Status:      domain.InstanceInProgress,
			SubmittedBy: "actor-123",
			SubmittedAt: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	h := &Handler{readSvc: fakeSvc}
	mux := getInstanceTestMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/approval/instances/inst-2", nil)
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "actor-1", []iamdomain.Role{}))
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if fakeSvc.gotTenantID != devTenantID {
		t.Fatalf("tenant_id = %q, want %q", fakeSvc.gotTenantID, devTenantID)
	}
}
