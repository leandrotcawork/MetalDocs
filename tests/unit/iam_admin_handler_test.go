package unit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	iamapp "metaldocs/internal/modules/iam/application"
	iamdelivery "metaldocs/internal/modules/iam/delivery/http"
	iammemory "metaldocs/internal/modules/iam/infrastructure/memory"
)

type fakeInvalidator struct {
	called bool
	userID string
}

func (f *fakeInvalidator) InvalidateUser(userID string) {
	f.called = true
	f.userID = userID
}

func TestIAMAdminHandlerUpsertRole(t *testing.T) {
	repo := iammemory.NewRoleAdminRepository()
	inv := &fakeInvalidator{}
	service := iamapp.NewAdminService(repo, inv)
	handler := iamdelivery.NewAdminHandler(service)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/iam/users/test-user/roles", strings.NewReader(`{"displayName":"Test User","role":"viewer"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin-local")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !inv.called || inv.userID != "test-user" {
		t.Fatalf("expected cache invalidation for test-user, got called=%v user=%s", inv.called, inv.userID)
	}
}
