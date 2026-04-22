package approvalhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterRoutes_AllRoutesRegistered(t *testing.T) {
	mux := http.NewServeMux()
	h := &Handler{}
	h.RegisterRoutes(mux)

	routes := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v2/documents/doc-1/submit"},
		{method: http.MethodPost, path: "/api/v2/approval/instances/i-1/stages/s-1/signoffs"},
		{method: http.MethodPost, path: "/api/v2/documents/doc-1/publish"},
		{method: http.MethodPost, path: "/api/v2/documents/doc-1/schedule-publish"},
		{method: http.MethodPost, path: "/api/v2/documents/doc-1/supersede"},
		{method: http.MethodPost, path: "/api/v2/documents/doc-1/obsolete"},
		{method: http.MethodPost, path: "/api/v2/approval/instances/i-1/cancel"},
		{method: http.MethodGet, path: "/api/v2/approval/instances/i-1"},
		{method: http.MethodGet, path: "/api/v2/approval/inbox"},
		{method: http.MethodPost, path: "/api/v2/approval/routes"},
		{method: http.MethodPut, path: "/api/v2/approval/routes/r-1"},
		{method: http.MethodDelete, path: "/api/v2/approval/routes/r-1"},
		{method: http.MethodGet, path: "/api/v2/approval/routes"},
	}

	for _, rt := range routes {
		req := httptest.NewRequest(rt.method, rt.path, nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code == http.StatusNotFound {
			t.Errorf("route %s %s not registered (got 404)", rt.method, rt.path)
		}
	}
}
