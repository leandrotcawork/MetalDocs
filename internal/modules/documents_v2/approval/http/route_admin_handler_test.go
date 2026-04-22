package approvalhttp

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"metaldocs/internal/modules/documents_v2/approval/application"
	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/http/contracts"
	"metaldocs/internal/modules/documents_v2/approval/repository"
	"metaldocs/internal/modules/iam/authz"
)

type fakeRouteAdminService struct {
	createResult application.CreateRouteResult
	createErr    error
	createReq    application.CreateRouteInput

	updateResult application.UpdateRouteResult
	updateErr    error
	updateReq    application.UpdateRouteInput

	deactivateResult application.DeactivateRouteResult
	deactivateErr    error
	deactivateReq    application.DeactivateRouteInput
}

func (f *fakeRouteAdminService) Create(_ context.Context, _ *sql.DB, in application.CreateRouteInput) (application.CreateRouteResult, error) {
	f.createReq = in
	if f.createErr != nil {
		return application.CreateRouteResult{}, f.createErr
	}
	return f.createResult, nil
}

func (f *fakeRouteAdminService) Update(_ context.Context, _ *sql.DB, in application.UpdateRouteInput) (application.UpdateRouteResult, error) {
	f.updateReq = in
	if f.updateErr != nil {
		return application.UpdateRouteResult{}, f.updateErr
	}
	return f.updateResult, nil
}

func (f *fakeRouteAdminService) Deactivate(_ context.Context, _ *sql.DB, in application.DeactivateRouteInput) (application.DeactivateRouteResult, error) {
	f.deactivateReq = in
	if f.deactivateErr != nil {
		return application.DeactivateRouteResult{}, f.deactivateErr
	}
	return f.deactivateResult, nil
}

func routeAdminTestMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v2/approval/routes", h.CreateRouteHandler)
	mux.HandleFunc("PUT /api/v2/approval/routes/{id}", h.UpdateRouteHandler)
	mux.HandleFunc("DELETE /api/v2/approval/routes/{id}", h.DeactivateRouteHandler)
	mux.HandleFunc("GET /api/v2/approval/routes", h.ListRoutesHandler)
	return mux
}

func TestCreateRoute_HappyPath(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "created"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeRouteAdminService{
				createResult: application.CreateRouteResult{RouteID: "route-123"},
			}
			h := &Handler{routeAdmin: svc}
			mux := routeAdminTestMux(h)

			body := `{"profile_code":"ops","name":"Ops Route","stages":[{"order":1,"name":"Review","required_role":"reviewer","required_capability":"doc.signoff","area_code":"ops","quorum":"any_1_of","drift_policy":"reduce_quorum"}]}`
			req := httptest.NewRequest(http.MethodPost, "/api/v2/approval/routes", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", "tenant-1")
			req.Header.Set("X-User-ID", "actor-1")
			req.Header.Set("Idempotency-Key", "idem-1")

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusCreated {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusCreated)
			}

			var out contracts.RouteResponse
			if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if out.RouteID != "route-123" {
				t.Fatalf("route_id = %q, want %q", out.RouteID, "route-123")
			}

			if svc.createReq.TenantID != "tenant-1" {
				t.Fatalf("tenant_id = %q, want %q", svc.createReq.TenantID, "tenant-1")
			}
			if svc.createReq.ActorUserID != "actor-1" {
				t.Fatalf("actor_user_id = %q, want %q", svc.createReq.ActorUserID, "actor-1")
			}
			if svc.createReq.ProfileCode != "ops" {
				t.Fatalf("profile_code = %q, want %q", svc.createReq.ProfileCode, "ops")
			}
			if svc.createReq.Name != "Ops Route" {
				t.Fatalf("name = %q, want %q", svc.createReq.Name, "Ops Route")
			}
			if len(svc.createReq.Stages) != 1 {
				t.Fatalf("stages len = %d, want 1", len(svc.createReq.Stages))
			}

			stage := svc.createReq.Stages[0]
			if stage.Order != 1 || stage.Name != "Review" || stage.RequiredRole != "reviewer" || stage.RequiredCapability != "doc.signoff" || stage.AreaCode != "ops" {
				t.Fatalf("unexpected stage mapping: %+v", stage)
			}
			if stage.Quorum != domain.QuorumPolicy("any_1_of") {
				t.Fatalf("stage quorum = %q, want %q", stage.Quorum, domain.QuorumPolicy("any_1_of"))
			}
			if stage.OnEligibilityDrift != domain.DriftPolicy("reduce_quorum") {
				t.Fatalf("stage drift policy = %q, want %q", stage.OnEligibilityDrift, domain.DriftPolicy("reduce_quorum"))
			}
		})
	}
}

func TestCreateRoute_CapDenied(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "capability denied"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeRouteAdminService{
				createErr: authz.ErrCapabilityDenied{Capability: "route.admin", AreaCode: "tenant", ActorID: "actor-1"},
			}
			h := &Handler{routeAdmin: svc}
			mux := routeAdminTestMux(h)

			body := `{"profile_code":"ops","name":"Ops Route","stages":[{"order":1,"name":"Review","required_role":"reviewer","required_capability":"doc.signoff","area_code":"ops","quorum":"any_1_of","drift_policy":"reduce_quorum"}]}`
			req := httptest.NewRequest(http.MethodPost, "/api/v2/approval/routes", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", "idem-1")

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
			}
		})
	}
}

func TestCreateRoute_DuplicateProfile(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "duplicate profile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeRouteAdminService{createErr: repository.ErrDuplicateRouteProfile}
			h := &Handler{routeAdmin: svc}
			mux := routeAdminTestMux(h)

			body := `{"profile_code":"ops","name":"Ops Route","stages":[{"order":1,"name":"Review","required_role":"reviewer","required_capability":"doc.signoff","area_code":"ops","quorum":"any_1_of","drift_policy":"reduce_quorum"}]}`
			req := httptest.NewRequest(http.MethodPost, "/api/v2/approval/routes", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", "idem-1")

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusConflict {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
			}
		})
	}
}

func TestUpdateRoute_HappyPath(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "updated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := 2
			svc := &fakeRouteAdminService{
				updateResult: application.UpdateRouteResult{RouteID: "route-1", NewVersion: 5},
			}
			h := &Handler{routeAdmin: svc}
			mux := routeAdminTestMux(h)

			body := `{"name":"Ops Route v2","stages":[{"order":1,"name":"Review","required_role":"reviewer","required_capability":"doc.signoff","area_code":"ops","quorum":"m_of_n","quorum_m":2,"drift_policy":"keep_snapshot"}]}`
			req := httptest.NewRequest(http.MethodPut, "/api/v2/approval/routes/route-1", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", "tenant-1")
			req.Header.Set("X-User-ID", "actor-1")
			req.Header.Set("Idempotency-Key", "idem-1")
			req.Header.Set("If-Match", "\"v4\"")

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
			}

			var out contracts.RouteResponse
			if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if out.RouteID != "route-1" {
				t.Fatalf("route_id = %q, want %q", out.RouteID, "route-1")
			}
			if out.NewVersion != 5 {
				t.Fatalf("new_version = %d, want %d", out.NewVersion, 5)
			}

			if svc.updateReq.TenantID != "tenant-1" || svc.updateReq.RouteID != "route-1" || svc.updateReq.ActorUserID != "actor-1" {
				t.Fatalf("unexpected request mapped to service: %+v", svc.updateReq)
			}
			if svc.updateReq.Name != "Ops Route v2" {
				t.Fatalf("name = %q, want %q", svc.updateReq.Name, "Ops Route v2")
			}
			if len(svc.updateReq.Stages) != 1 {
				t.Fatalf("stages len = %d, want 1", len(svc.updateReq.Stages))
			}
			if svc.updateReq.Stages[0].Quorum != domain.QuorumPolicy("m_of_n") {
				t.Fatalf("stage quorum = %q, want %q", svc.updateReq.Stages[0].Quorum, domain.QuorumPolicy("m_of_n"))
			}
			if svc.updateReq.Stages[0].QuorumM == nil || *svc.updateReq.Stages[0].QuorumM != m {
				t.Fatalf("stage quorum_m = %#v, want %d", svc.updateReq.Stages[0].QuorumM, m)
			}
		})
	}
}

func TestUpdateRoute_RouteInUse(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "route in use"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeRouteAdminService{updateErr: repository.ErrRouteInUse}
			h := &Handler{routeAdmin: svc}
			mux := routeAdminTestMux(h)

			body := `{"name":"Ops Route v2","stages":[{"order":1,"name":"Review","required_role":"reviewer","required_capability":"doc.signoff","area_code":"ops","quorum":"all_of","drift_policy":"fail_stage"}]}`
			req := httptest.NewRequest(http.MethodPut, "/api/v2/approval/routes/route-1", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", "idem-1")
			req.Header.Set("If-Match", "\"v4\"")

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusConflict {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
			}
		})
	}
}

func TestDeactivateRoute_HappyPath(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "deactivated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeRouteAdminService{
				deactivateResult: application.DeactivateRouteResult{RouteID: "route-1"},
			}
			h := &Handler{routeAdmin: svc}
			mux := routeAdminTestMux(h)

			req := httptest.NewRequest(http.MethodDelete, "/api/v2/approval/routes/route-1", nil)
			req.Header.Set("X-Tenant-ID", "tenant-1")
			req.Header.Set("X-User-ID", "actor-1")
			req.Header.Set("Idempotency-Key", "idem-1")
			req.Header.Set("If-Match", "\"v3\"")

			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
			}

			var out contracts.RouteResponse
			if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if out.RouteID != "route-1" {
				t.Fatalf("route_id = %q, want %q", out.RouteID, "route-1")
			}
			if svc.deactivateReq.TenantID != "tenant-1" || svc.deactivateReq.RouteID != "route-1" || svc.deactivateReq.ActorUserID != "actor-1" {
				t.Fatalf("unexpected request mapped to service: %+v", svc.deactivateReq)
			}
		})
	}
}
