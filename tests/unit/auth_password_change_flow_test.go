package unit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authapp "metaldocs/internal/modules/auth/application"
	httpdelivery "metaldocs/internal/modules/auth/delivery/http"
	authmemory "metaldocs/internal/modules/auth/infrastructure/memory"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

func TestPasswordChangePreservesSessionAndClearsMustChangePassword(t *testing.T) {
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
	if err := svc.CreateUser(context.Background(), "flow-user", "flow.user", "flow.user@test.local", "Flow User", "abc12345", []iamdomain.Role{iamdomain.RoleViewer}, "test"); err != nil {
		t.Fatalf("create user: %v", err)
	}

	authHandler := httpdelivery.NewHandler(svc)
	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)
	handler := httpdelivery.NewMiddleware(svc, cfg, true).Wrap(mux)

	loginResp := performJSONRequest(t, handler, http.MethodPost, "/api/v1/auth/login", `{"identifier":"flow.user","password":"abc12345"}`, nil)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected initial login 200, got %d body=%s", loginResp.Code, loginResp.Body.String())
	}
	loginPayload := decodeMap(t, loginResp.Body.String())
	userPayload := loginPayload["user"].(map[string]any)
	if userPayload["mustChangePassword"] != true {
		t.Fatalf("expected mustChangePassword=true on first login, got %#v", userPayload["mustChangePassword"])
	}

	sessionCookie := findCookie(t, loginResp.Result().Cookies(), cfg.SessionCookieName)
	changeResp := performJSONRequest(t, handler, http.MethodPost, "/api/v1/auth/change-password", `{"newPassword":"abc12346"}`, sessionCookie)
	if changeResp.Code != http.StatusOK {
		t.Fatalf("expected change password 200, got %d body=%s", changeResp.Code, changeResp.Body.String())
	}

	changePayload := decodeMap(t, changeResp.Body.String())
	rotatedUser := changePayload["user"].(map[string]any)
	if rotatedUser["mustChangePassword"] != false {
		t.Fatalf("expected mustChangePassword=false after password change, got %#v", rotatedUser["mustChangePassword"])
	}

	meResp := performJSONRequest(t, handler, http.MethodGet, "/api/v1/auth/me", "", sessionCookie)
	if meResp.Code != http.StatusOK {
		t.Fatalf("expected existing session to remain authorized, got %d body=%s", meResp.Code, meResp.Body.String())
	}

	oldLoginResp := performJSONRequest(t, handler, http.MethodPost, "/api/v1/auth/login", `{"identifier":"flow.user","password":"abc12345"}`, nil)
	if oldLoginResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected old password login to fail with 401, got %d body=%s", oldLoginResp.Code, oldLoginResp.Body.String())
	}

	newLoginResp := performJSONRequest(t, handler, http.MethodPost, "/api/v1/auth/login", `{"identifier":"flow.user","password":"abc12346"}`, nil)
	if newLoginResp.Code != http.StatusOK {
		t.Fatalf("expected new password login 200, got %d body=%s", newLoginResp.Code, newLoginResp.Body.String())
	}
	newLoginPayload := decodeMap(t, newLoginResp.Body.String())
	newUserPayload := newLoginPayload["user"].(map[string]any)
	if newUserPayload["mustChangePassword"] != false {
		t.Fatalf("expected mustChangePassword=false after relogin, got %#v", newUserPayload["mustChangePassword"])
	}
}

func performJSONRequest(t *testing.T, handler http.Handler, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func findCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %s not found", name)
	return nil
}

func decodeMap(t *testing.T, body string) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("decode json: %v body=%s", err, body)
	}
	return payload
}
