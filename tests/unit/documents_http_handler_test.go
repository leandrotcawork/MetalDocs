package unit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"metaldocs/internal/modules/documents/application"
	httpdelivery "metaldocs/internal/modules/documents/delivery/http"
	"metaldocs/internal/modules/documents/infrastructure/memory"
)

func newTestMux() *http.ServeMux {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	h := httpdelivery.NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func TestHealthEndpoints(t *testing.T) {
	mux := newTestMux()

	for _, path := range []string{"/api/v1/health/live", "/api/v1/health/ready"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", path, rr.Code)
		}
	}
}

func TestCreateAndListVersionsFlow(t *testing.T) {
	mux := newTestMux()

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/documents", strings.NewReader(`{"title":"Contract","documentType":"contract","ownerId":"u1","businessUnit":"legal","department":"contracts","classification":"INTERNAL"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	mux.ServeHTTP(createRR, createReq)

	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createRR.Code)
	}

	var created map[string]any
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil {
		t.Fatalf("invalid create response json: %v", err)
	}

	documentID, ok := created["documentId"].(string)
	if !ok || strings.TrimSpace(documentID) == "" {
		t.Fatal("expected non-empty documentId")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+documentID+"/versions", nil)
	listRR := httptest.NewRecorder()
	mux.ServeHTTP(listRR, listReq)

	if listRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listRR.Code)
	}
}

func TestCreateDocumentValidationError(t *testing.T) {
	mux := newTestMux()

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/documents", strings.NewReader(`{"title":"","documentType":"","ownerId":"","businessUnit":"","department":""}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	mux.ServeHTTP(createRR, createReq)

	if createRR.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", createRR.Code)
	}
}

func TestListDocumentTypes(t *testing.T) {
	mux := newTestMux()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/document-types", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
