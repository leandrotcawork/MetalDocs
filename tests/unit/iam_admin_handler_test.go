package unit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authapp "metaldocs/internal/modules/auth/application"
	authdomain "metaldocs/internal/modules/auth/domain"
	authmemory "metaldocs/internal/modules/auth/infrastructure/memory"
	iamapp "metaldocs/internal/modules/iam/application"
	iamdelivery "metaldocs/internal/modules/iam/delivery/http"
	iamdomain "metaldocs/internal/modules/iam/domain"
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
	handler := iamdelivery.NewAdminHandler(service, nil)

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

func TestIAMAdminHandlerReplaceRoles(t *testing.T) {
	repo := iammemory.NewRoleAdminRepository()
	inv := &fakeInvalidator{}
	service := iamapp.NewAdminService(repo, inv)
	handler := iamdelivery.NewAdminHandler(service, nil)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/iam/users/test-user/roles", strings.NewReader(`{"displayName":"Test User","roles":["editor","reviewer"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin-local")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !inv.called || inv.userID != "test-user" {
		t.Fatalf("expected cache invalidation for test-user, got called=%v user=%s", inv.called, inv.userID)
	}
}

func TestIAMAdminHandlerResetPassword(t *testing.T) {
	repo := iammemory.NewRoleAdminRepository()
	inv := &fakeInvalidator{}
	service := iamapp.NewAdminService(repo, inv)
	authRepo := authmemory.NewRepository()
	authService := authapp.NewService(authRepo, iamapp.NewDevRoleProvider(map[string][]iamdomain.Role{
		"test-user": {iamdomain.RoleViewer},
	}), authapp.Config{
		PasswordMinLength:      8,
		LoginMaxFailedAttempts: 5,
		LoginLockDuration:      time.Minute,
	})
	if err := authRepo.CreateUser(context.Background(), authdomain.CreateUserParams{
		UserID:             "test-user",
		Username:           "test-user",
		DisplayName:        "Test User",
		PasswordHash:       "$2a$10$W8v2u5Q2m4B8T7d7YQxM5eK/94B2SxA4yJ1mQ1MNhbbX6YsJhBKtC",
		PasswordAlgo:       "bcrypt",
		MustChangePassword: false,
		IsActive:           true,
		Roles:              []iamdomain.Role{iamdomain.RoleViewer},
		CreatedBy:          "system",
	}); err != nil {
		t.Fatalf("seed auth user: %v", err)
	}
	handler := iamdelivery.NewAdminHandler(service, authService)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/iam/users/test-user/reset-password", strings.NewReader(`{"newPassword":"Reset1234"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin-local")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	identity, err := authRepo.FindIdentityByUserID(context.Background(), "test-user")
	if err != nil {
		t.Fatalf("load user: %v", err)
	}
	if !identity.MustChangePassword {
		t.Fatalf("expected must change password to be true after reset")
	}
}

func TestIAMAdminHandlerUnlockUser(t *testing.T) {
	repo := iammemory.NewRoleAdminRepository()
	inv := &fakeInvalidator{}
	service := iamapp.NewAdminService(repo, inv)
	authRepo := authmemory.NewRepository()
	authService := authapp.NewService(authRepo, iamapp.NewDevRoleProvider(map[string][]iamdomain.Role{
		"test-user": {iamdomain.RoleViewer},
	}), authapp.Config{
		PasswordMinLength:      8,
		LoginMaxFailedAttempts: 5,
		LoginLockDuration:      time.Minute,
	})
	if err := authRepo.CreateUser(context.Background(), authdomain.CreateUserParams{
		UserID:             "test-user",
		Username:           "test-user",
		DisplayName:        "Test User",
		PasswordHash:       "$2a$10$W8v2u5Q2m4B8T7d7YQxM5eK/94B2SxA4yJ1mQ1MNhbbX6YsJhBKtC",
		PasswordAlgo:       "bcrypt",
		MustChangePassword: false,
		IsActive:           true,
		Roles:              []iamdomain.Role{iamdomain.RoleViewer},
		CreatedBy:          "system",
	}); err != nil {
		t.Fatalf("seed auth user: %v", err)
	}
	lock := time.Now().UTC().Add(time.Minute)
	if err := authRepo.RecordFailedLogin(context.Background(), "test-user", 5, &lock); err != nil {
		t.Fatalf("seed lock state: %v", err)
	}
	handler := iamdelivery.NewAdminHandler(service, authService)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/iam/users/test-user/unlock", nil)
	req.Header.Set("X-User-Id", "admin-local")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	identity, err := authRepo.FindIdentityByUserID(context.Background(), "test-user")
	if err != nil {
		t.Fatalf("load user: %v", err)
	}
	if identity.FailedLoginAttempts != 0 {
		t.Fatalf("expected failed attempts to be reset, got %d", identity.FailedLoginAttempts)
	}
	if identity.LockedUntil != nil {
		t.Fatalf("expected user to be unlocked")
	}
}
