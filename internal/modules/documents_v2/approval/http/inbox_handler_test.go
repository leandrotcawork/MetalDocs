package approvalhttp

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"metaldocs/internal/modules/documents_v2/approval/domain"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

type fakeReadServiceInbox struct {
	items []domain.Instance
	err   error

	called      bool
	gotTenantID string
	gotActorID  string
	gotAreaCode string
	gotLimit    int
	gotOffset   int
}

func (f *fakeReadServiceInbox) LoadInstance(_ context.Context, _ *sql.DB, _, _, _ string) (*domain.Instance, error) {
	return nil, nil
}

func (f *fakeReadServiceInbox) LoadActiveInstanceByDocument(_ context.Context, _ *sql.DB, _, _ string) (*domain.Instance, error) {
	return nil, nil
}

func (f *fakeReadServiceInbox) ListPendingForActor(_ context.Context, _ *sql.DB, tenantID, actorID string, areaCode string, limit, offset int) ([]domain.Instance, error) {
	f.called = true
	f.gotTenantID = tenantID
	f.gotActorID = actorID
	f.gotAreaCode = areaCode
	f.gotLimit = limit
	f.gotOffset = offset
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

func inboxTestMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v2/approval/inbox", h.InboxHandler)
	return mux
}

func TestInboxHandler_HappyEmptyList(t *testing.T) {
	fakeSvc := &fakeReadServiceInbox{items: []domain.Instance{}}
	h := &Handler{readSvc: fakeSvc}
	mux := inboxTestMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/approval/inbox", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "actor-1", []iamdomain.Role{}))
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !fakeSvc.called {
		t.Fatalf("expected read service call")
	}
	if fakeSvc.gotLimit != 25 || fakeSvc.gotOffset != 0 {
		t.Fatalf("unexpected defaults limit=%d offset=%d", fakeSvc.gotLimit, fakeSvc.gotOffset)
	}
}

func TestInboxHandler_ValidLimitParam(t *testing.T) {
	fakeSvc := &fakeReadServiceInbox{}
	h := &Handler{readSvc: fakeSvc}
	mux := inboxTestMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/approval/inbox?area_code=finance&limit=40&offset=10", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "actor-1", []iamdomain.Role{}))
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if fakeSvc.gotAreaCode != "finance" {
		t.Fatalf("area_code = %q, want %q", fakeSvc.gotAreaCode, "finance")
	}
	if fakeSvc.gotLimit != 40 || fakeSvc.gotOffset != 10 {
		t.Fatalf("limit/offset = %d/%d, want %d/%d", fakeSvc.gotLimit, fakeSvc.gotOffset, 40, 10)
	}
}

func TestInboxHandler_InvalidLimitTooLarge(t *testing.T) {
	fakeSvc := &fakeReadServiceInbox{}
	h := &Handler{readSvc: fakeSvc}
	mux := inboxTestMux(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/approval/inbox?limit=101", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "actor-1", []iamdomain.Role{}))
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if fakeSvc.called {
		t.Fatalf("service should not be called on invalid limit")
	}
}
