package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	auditmemory "metaldocs/internal/modules/audit/infrastructure/memory"
	docapp "metaldocs/internal/modules/documents/application"
	docdelivery "metaldocs/internal/modules/documents/delivery/http"
	memoryrepo "metaldocs/internal/modules/documents/infrastructure/memory"
	iamapp "metaldocs/internal/modules/iam/application"
	iamdelivery "metaldocs/internal/modules/iam/delivery/http"
	iamdomain "metaldocs/internal/modules/iam/domain"
	iammemory "metaldocs/internal/modules/iam/infrastructure/memory"
	searchapp "metaldocs/internal/modules/search/application"
	searchdelivery "metaldocs/internal/modules/search/delivery/http"
	searchdocs "metaldocs/internal/modules/search/infrastructure/documents"
	workflowapp "metaldocs/internal/modules/workflow/application"
	workflowdelivery "metaldocs/internal/modules/workflow/delivery/http"
	"metaldocs/internal/platform/config"
	"metaldocs/internal/platform/observability"
	"metaldocs/internal/platform/security"
)

func TestAPIContractSmoke(t *testing.T) {
	handler := buildContractTestHandler()

	docID := createDocument(t, handler)

	cases := []struct {
		name       string
		method     string
		path       string
		body       any
		withUserID bool
		wantStatus int
	}{
		{name: "health live", method: http.MethodGet, path: "/api/v1/health/live", wantStatus: http.StatusOK},
		{name: "health ready", method: http.MethodGet, path: "/api/v1/health/ready", wantStatus: http.StatusOK},
		{name: "metrics", method: http.MethodGet, path: "/api/v1/metrics", wantStatus: http.StatusOK},
		{name: "list document types", method: http.MethodGet, path: "/api/v1/document-types", withUserID: true, wantStatus: http.StatusOK},
		{name: "list access policies", method: http.MethodGet, path: "/api/v1/access-policies?resourceScope=document&resourceId=" + docID, withUserID: true, wantStatus: http.StatusOK},
		{name: "list documents", method: http.MethodGet, path: "/api/v1/documents", withUserID: true, wantStatus: http.StatusOK},
		{name: "get document", method: http.MethodGet, path: "/api/v1/documents/" + docID, withUserID: true, wantStatus: http.StatusOK},
		{name: "search documents", method: http.MethodGet, path: "/api/v1/search/documents?limit=10&documentType=contract&businessUnit=legal&department=contracts", withUserID: true, wantStatus: http.StatusOK},
		{
			name:       "replace access policies",
			method:     http.MethodPut,
			path:       "/api/v1/access-policies",
			body:       map[string]any{"resourceScope": "document", "resourceId": docID, "policies": []map[string]any{{"subjectType": "role", "subjectId": "admin", "capability": "document.view", "effect": "allow"}}},
			withUserID: true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "workflow transition",
			method:     http.MethodPost,
			path:       "/api/v1/workflow/documents/" + docID + "/transitions",
			body:       map[string]any{"toStatus": "IN_REVIEW", "reason": "contract-smoke"},
			withUserID: true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "list versions",
			method:     http.MethodGet,
			path:       "/api/v1/documents/" + docID + "/versions",
			withUserID: true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "upsert iam role",
			method:     http.MethodPost,
			path:       "/api/v1/iam/users/contract-user/roles",
			body:       map[string]any{"displayName": "Contract User", "role": "editor"},
			withUserID: true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing auth header",
			method:     http.MethodGet,
			path:       "/api/v1/documents",
			withUserID: false,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var bodyBytes []byte
			if tc.body != nil {
				var err error
				bodyBytes, err = json.Marshal(tc.body)
				if err != nil {
					t.Fatalf("marshal body: %v", err)
				}
			}

			req := httptest.NewRequest(tc.method, tc.path, bytes.NewReader(bodyBytes))
			if tc.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			if tc.withUserID {
				req.Header.Set("X-User-Id", "admin-local")
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("status mismatch for %s %s: got=%d want=%d body=%s", tc.method, tc.path, rr.Code, tc.wantStatus, rr.Body.String())
			}
		})
	}
}

func buildContractTestHandler() http.Handler {
	docRepo := memoryrepo.NewRepository()
	cachedProvider := iamapp.NewCachedRoleProvider(
		iamapp.NewDevRoleProvider(map[string][]iamdomain.Role{
			"admin-local": {iamdomain.RoleAdmin},
		}),
		0,
	)
	roleAdminRepo := iammemory.NewRoleAdminRepository()
	auditWriter := auditmemory.NewWriter()
	authorizer := iamapp.NewStaticAuthorizer()

	docService := docapp.NewService(docRepo, nil, nil)
	docHandler := docdelivery.NewHandler(docService)
	searchService := searchapp.NewService(searchdocs.NewReader(docRepo))
	searchHandler := searchdelivery.NewHandler(searchService)
	workflowService := workflowapp.NewService(docRepo, auditWriter, nil, nil)
	workflowHandler := workflowdelivery.NewHandler(workflowService)
	iamAdminService := iamapp.NewAdminService(roleAdminRepo, cachedProvider)
	iamAdminHandler := iamdelivery.NewAdminHandler(iamAdminService)
	iamMiddleware := iamdelivery.NewMiddleware(authorizer, cachedProvider, true)
	httpObs := observability.NewHTTPObservability()
	rateLimiter := security.NewRateLimiter(config.RateLimitConfig{Enabled: false})

	mux := http.NewServeMux()
	docHandler.RegisterRoutes(mux)
	searchHandler.RegisterRoutes(mux)
	workflowHandler.RegisterRoutes(mux)
	iamAdminHandler.RegisterRoutes(mux)
	mux.Handle("/api/v1/metrics", httpObs.MetricsHandler())

	return httpObs.Wrap(rateLimiter.Wrap(iamMiddleware.Wrap(mux)))
}

func createDocument(t *testing.T, handler http.Handler) string {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"title":          "Contract Seed",
		"documentType":   "contract",
		"ownerId":        "owner-contract",
		"businessUnit":   "legal",
		"department":     "contracts",
		"classification": "INTERNAL",
		"initialContent": "seed",
	})
	if err != nil {
		t.Fatalf("marshal create payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin-local")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("seed create status mismatch: got=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		DocumentID string `json:"documentId"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if resp.DocumentID == "" {
		t.Fatalf("seed create returned empty documentId")
	}
	return resp.DocumentID
}
