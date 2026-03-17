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

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/documents", strings.NewReader(`{"title":"Contract","documentType":"contract","ownerId":"u1","businessUnit":"legal","department":"contracts","classification":"INTERNAL","metadata":{"counterparty":"Metal Nobre","contract_number":"CNT-002","start_date":"2026-03-01","end_date":"2026-12-31"}}`))
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

	addReq := httptest.NewRequest(http.MethodPost, "/api/v1/documents/"+documentID+"/versions", strings.NewReader(`{"content":"v2","changeSummary":"updated body"}`))
	addReq.Header.Set("Content-Type", "application/json")
	addRR := httptest.NewRecorder()
	mux.ServeHTTP(addRR, addReq)

	if addRR.Code != http.StatusCreated {
		t.Fatalf("expected 201 for add version, got %d", addRR.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+documentID, nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)

	if getRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for get document, got %d", getRR.Code)
	}

	diffReq := httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+documentID+"/versions/diff?fromVersion=1&toVersion=2", nil)
	diffRR := httptest.NewRecorder()
	mux.ServeHTTP(diffRR, diffReq)

	if diffRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for diff, got %d", diffRR.Code)
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

func TestReplaceAndListAccessPoliciesHTTP(t *testing.T) {
	mux := newTestMux()

	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/access-policies", strings.NewReader(`{"resourceScope":"document","resourceId":"doc-1","policies":[{"subjectType":"user","subjectId":"leandro","capability":"document.view","effect":"allow"}]}`))
	putReq.Header.Set("Content-Type", "application/json")
	putRR := httptest.NewRecorder()
	mux.ServeHTTP(putRR, putReq)

	if putRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", putRR.Code, putRR.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/access-policies?resourceScope=document&resourceId=doc-1", nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)

	if getRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", getRR.Code, getRR.Body.String())
	}
}
