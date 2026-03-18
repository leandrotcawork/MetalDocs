package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"metaldocs/internal/platform/security"
)

func TestOriginProtectionAllowsTrustedOriginWithSessionCookie(t *testing.T) {
	protector := security.NewOriginProtection(security.OriginProtectionConfig{
		Enabled:           true,
		SessionCookieName: "metaldocs_session",
		TrustedOrigins:    []string{"http://127.0.0.1:4173"},
	})

	handler := protector.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/api/v1/auth/change-password", nil)
	req.AddCookie(&http.Cookie{Name: "metaldocs_session", Value: "session-token"})
	req.Header.Set("Origin", "http://127.0.0.1:4173")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected trusted origin request to pass, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestOriginProtectionBlocksUnsafeRequestWithoutOrigin(t *testing.T) {
	protector := security.NewOriginProtection(security.OriginProtectionConfig{
		Enabled:           true,
		SessionCookieName: "metaldocs_session",
		TrustedOrigins:    []string{"http://127.0.0.1:4173"},
	})

	handler := protector.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/api/v1/auth/change-password", nil)
	req.AddCookie(&http.Cookie{Name: "metaldocs_session", Value: "session-token"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected missing origin to be rejected, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestOriginProtectionAllowsGetWithoutOrigin(t *testing.T) {
	protector := security.NewOriginProtection(security.OriginProtectionConfig{
		Enabled:           true,
		SessionCookieName: "metaldocs_session",
		TrustedOrigins:    []string{"http://127.0.0.1:4173"},
	})

	handler := protector.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "metaldocs_session", Value: "session-token"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected safe method to bypass origin protection, got %d body=%s", rr.Code, rr.Body.String())
	}
}
