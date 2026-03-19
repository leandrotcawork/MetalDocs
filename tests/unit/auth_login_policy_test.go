package unit

import (
	"context"
	"net/http"
	"testing"
	"time"

	authapp "metaldocs/internal/modules/auth/application"
	authdomain "metaldocs/internal/modules/auth/domain"
	authmemory "metaldocs/internal/modules/auth/infrastructure/memory"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

func TestAuthenticateLocksAfterRepeatedFailures(t *testing.T) {
	repo := authmemory.NewRepository()
	cfg := authapp.Config{
		SessionCookieName:      "metaldocs_session",
		SessionTTL:             time.Hour,
		SessionSecret:          "local-test-secret",
		PasswordMinLength:      8,
		LoginMaxFailedAttempts: 3,
		LoginLockDuration:      5 * time.Minute,
	}
	svc := authapp.NewService(repo, repo, repo, cfg)
	if err := svc.CreateUser(context.Background(), "lock-user", "lock.user", "lock.user@test.local", "Lock User", "abc12345", []iamdomain.Role{iamdomain.RoleViewer}, "test"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	req := httptestRequest(http.MethodPost, "/api/v1/auth/login")
	for i := 0; i < 3; i++ {
		if _, err := svc.Authenticate(context.Background(), "lock.user", "wrong-pass", req); err == nil {
			t.Fatalf("expected invalid credentials on attempt %d", i+1)
		}
	}

	if _, err := svc.Authenticate(context.Background(), "lock.user", "abc12345", req); err != authdomain.ErrIdentityLocked {
		t.Fatalf("expected account lock after repeated failures, got %v", err)
	}
}

func TestAuthenticateRejectsInactiveUser(t *testing.T) {
	repo := authmemory.NewRepository()
	cfg := authapp.Config{
		SessionCookieName:      "metaldocs_session",
		SessionTTL:             time.Hour,
		SessionSecret:          "local-test-secret",
		PasswordMinLength:      8,
		LoginMaxFailedAttempts: 5,
		LoginLockDuration:      5 * time.Minute,
	}
	svc := authapp.NewService(repo, repo, repo, cfg)
	if err := svc.CreateUser(context.Background(), "inactive-user", "inactive.user", "inactive.user@test.local", "Inactive User", "abc12345", []iamdomain.Role{iamdomain.RoleViewer}, "test"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	inactive := false
	if err := svc.UpdateUser(context.Background(), authdomain.UpdateUserParams{
		UserID:   "inactive-user",
		IsActive: &inactive,
	}, ""); err != nil {
		t.Fatalf("deactivate user: %v", err)
	}

	if _, err := svc.Authenticate(context.Background(), "inactive.user", "abc12345", httptestRequest(http.MethodPost, "/api/v1/auth/login")); err != authdomain.ErrIdentityInactive {
		t.Fatalf("expected inactive account error, got %v", err)
	}
}

func httptestRequest(method, path string) *http.Request {
	req, _ := http.NewRequest(method, path, nil)
	req.RemoteAddr = "127.0.0.1:12345"
	return req
}
