package contract

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
	auditapp "metaldocs/internal/modules/audit/application"
	auditdelivery "metaldocs/internal/modules/audit/delivery/http"
	auditmemory "metaldocs/internal/modules/audit/infrastructure/memory"
	authapp "metaldocs/internal/modules/auth/application"
	authdomain "metaldocs/internal/modules/auth/domain"
	authmemory "metaldocs/internal/modules/auth/infrastructure/memory"
	docapp "metaldocs/internal/modules/documents/application"
	docdelivery "metaldocs/internal/modules/documents/delivery/http"
	memoryrepo "metaldocs/internal/modules/documents/infrastructure/memory"
	iamapp "metaldocs/internal/modules/iam/application"
	iamdelivery "metaldocs/internal/modules/iam/delivery/http"
	iamdomain "metaldocs/internal/modules/iam/domain"
	iammemory "metaldocs/internal/modules/iam/infrastructure/memory"
	notificationapp "metaldocs/internal/modules/notifications/application"
	notificationdelivery "metaldocs/internal/modules/notifications/delivery/http"
	notificationdomain "metaldocs/internal/modules/notifications/domain"
	notificationmemory "metaldocs/internal/modules/notifications/infrastructure/memory"
	searchapp "metaldocs/internal/modules/search/application"
	searchdelivery "metaldocs/internal/modules/search/delivery/http"
	searchdocs "metaldocs/internal/modules/search/infrastructure/documents"
	workflowapp "metaldocs/internal/modules/workflow/application"
	workflowdelivery "metaldocs/internal/modules/workflow/delivery/http"
	workflowmemory "metaldocs/internal/modules/workflow/infrastructure/memory"
	"metaldocs/internal/platform/config"
	"metaldocs/internal/platform/observability"
	"metaldocs/internal/platform/security"
)

func TestOpenAPIContainsSchemaRuntimeEndpoints(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "api", "openapi", "v1", "openapi.yaml"))
	if err != nil {
		t.Fatalf("read openapi: %v", err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		t.Fatalf("parse openapi yaml: %v", err)
	}

	paths := findMappingValue(&root, "paths")
	if paths == nil {
		t.Fatal("missing paths section")
	}

	required := map[string]string{
		"/document-types/{typeKey}/bundle":      "get",
		"/documents/{documentId}/editor-bundle": "get",
		"/documents/{documentId}/content":       "put",
		"/documents/{documentId}/export/docx":   "post",
	}

	for path, method := range required {
		pathNode := findMappingValue(paths, path)
		if pathNode == nil {
			t.Fatalf("missing path %s", path)
		}
		if findMappingValue(pathNode, method) == nil {
			t.Fatalf("missing method %s for path %s", method, path)
		}
	}
}

func findMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

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
		{name: "metrics", method: http.MethodGet, path: "/api/v1/metrics", withUserID: true, wantStatus: http.StatusOK},
		{name: "list document families", method: http.MethodGet, path: "/api/v1/document-families", withUserID: true, wantStatus: http.StatusOK},
		{name: "list document profiles", method: http.MethodGet, path: "/api/v1/document-profiles", withUserID: true, wantStatus: http.StatusOK},
		{name: "list profile schema", method: http.MethodGet, path: "/api/v1/document-profiles/it/schema", withUserID: true, wantStatus: http.StatusOK},
		{name: "get profile governance", method: http.MethodGet, path: "/api/v1/document-profiles/it/governance", withUserID: true, wantStatus: http.StatusOK},
		{name: "list process areas", method: http.MethodGet, path: "/api/v1/process-areas", withUserID: true, wantStatus: http.StatusOK},
		{name: "list document subjects", method: http.MethodGet, path: "/api/v1/document-subjects", withUserID: true, wantStatus: http.StatusOK},
		{name: "list document types", method: http.MethodGet, path: "/api/v1/document-types", withUserID: true, wantStatus: http.StatusOK},
		{name: "list access policies", method: http.MethodGet, path: "/api/v1/access-policies?resourceScope=document&resourceId=" + docID, withUserID: true, wantStatus: http.StatusOK},
		{name: "list documents", method: http.MethodGet, path: "/api/v1/documents", withUserID: true, wantStatus: http.StatusOK},
		{name: "get document", method: http.MethodGet, path: "/api/v1/documents/" + docID, withUserID: true, wantStatus: http.StatusOK},
		{name: "search documents", method: http.MethodGet, path: "/api/v1/search/documents?limit=10&documentProfile=po&processArea=marketplaces&businessUnit=commercial&department=marketplaces", withUserID: true, wantStatus: http.StatusOK},
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
			body:       map[string]any{"toStatus": "IN_REVIEW", "reason": "contract-smoke", "assignedReviewer": "admin-local"},
			withUserID: true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "list workflow approvals",
			method:     http.MethodGet,
			path:       "/api/v1/workflow/documents/" + docID + "/approvals",
			withUserID: true,
			wantStatus: http.StatusOK,
		},
		{name: "list audit events", method: http.MethodGet, path: "/api/v1/audit/events?resourceType=document&resourceId=" + docID + "&limit=20", withUserID: true, wantStatus: http.StatusOK},
		{name: "list notifications", method: http.MethodGet, path: "/api/v1/notifications?limit=20", withUserID: true, wantStatus: http.StatusOK},
		{name: "mark notification read", method: http.MethodPost, path: "/api/v1/notifications/notif-contract/read", withUserID: true, wantStatus: http.StatusOK},
		{
			name:       "list versions",
			method:     http.MethodGet,
			path:       "/api/v1/documents/" + docID + "/versions",
			withUserID: true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "list attachments",
			method:     http.MethodGet,
			path:       "/api/v1/documents/" + docID + "/attachments",
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
			name:       "replace iam roles",
			method:     http.MethodPut,
			path:       "/api/v1/iam/users/contract-user/roles",
			body:       map[string]any{"displayName": "Contract User", "roles": []string{"reviewer", "viewer"}},
			withUserID: true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "admin reset password",
			method:     http.MethodPost,
			path:       "/api/v1/iam/users/contract-user/reset-password",
			body:       map[string]any{"newPassword": "ContractReset123"},
			withUserID: true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "admin unlock user",
			method:     http.MethodPost,
			path:       "/api/v1/iam/users/contract-user/unlock",
			withUserID: true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "metrics without auth",
			method:     http.MethodGet,
			path:       "/api/v1/metrics",
			withUserID: false,
			wantStatus: http.StatusUnauthorized,
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
			if tc.path == "/api/v1/metrics" && tc.wantStatus == http.StatusOK {
				var payload map[string]any
				if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
					t.Fatalf("invalid metrics json: %v", err)
				}
				if _, ok := payload["runtime"]; !ok {
					t.Fatalf("expected runtime metrics payload, got %s", rr.Body.String())
				}
			}
			if tc.path == "/api/v1/health/ready" {
				var payload map[string]any
				if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
					t.Fatalf("invalid readiness json: %v", err)
				}
				if payload["status"] != "ready" {
					t.Fatalf("expected ready status, got %v", payload["status"])
				}
			}
			if tc.path == "/api/v1/document-profiles" {
				var payload struct {
					Items []struct {
						Code  string `json:"code"`
						Alias string `json:"alias"`
					} `json:"items"`
				}
				if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
					t.Fatalf("invalid document profiles json: %v", err)
				}
				if len(payload.Items) == 0 {
					t.Fatal("expected document profiles payload")
				}
				for _, item := range payload.Items {
					if item.Alias == "" {
						t.Fatalf("expected alias for profile %s", item.Code)
					}
				}
			}
		})
	}
}

func buildContractTestHandler() http.Handler {
	docRepo := memoryrepo.NewRepository()
	attachmentStore := memoryrepo.NewAttachmentStore()
	cachedProvider := iamapp.NewCachedRoleProvider(
		iamapp.NewDevRoleProvider(map[string][]iamdomain.Role{
			"admin-local": {iamdomain.RoleAdmin},
		}),
		0,
	)
	roleAdminRepo := iammemory.NewRoleAdminRepository()
	auditStore := auditmemory.NewWriter()
	authorizer := iamapp.NewStaticAuthorizer()
	notifRepo := notificationmemory.NewRepository()
	authRepo := authmemory.NewRepository()

	auditService := auditapp.NewService(auditStore)
	docService := docapp.NewService(docRepo, nil, nil).WithAttachmentStore(attachmentStore)
	auditHandler := auditdelivery.NewHandler(auditService)
	docHandler := docdelivery.NewHandler(docService)
	notificationService := notificationapp.NewService(notifRepo, docRepo, nil)
	notificationHandler := notificationdelivery.NewHandler(notificationService)
	searchService := searchapp.NewService(searchdocs.NewReader(docRepo))
	searchHandler := searchdelivery.NewHandler(searchService)
	workflowService := workflowapp.NewService(docRepo, workflowmemory.NewApprovalRepository(), auditStore, nil, nil)
	workflowHandler := workflowdelivery.NewHandler(workflowService)
	iamAdminService := iamapp.NewAdminService(roleAdminRepo, cachedProvider)
	iamAdminHandler := iamdelivery.NewAdminHandler(iamAdminService, authapp.NewService(authRepo, cachedProvider, roleAdminRepo, authapp.Config{PasswordMinLength: 8, LoginMaxFailedAttempts: 5, LoginLockDuration: time.Minute}), auditStore)
	iamMiddleware := iamdelivery.NewMiddleware(authorizer, cachedProvider, true, true)
	statusProvider := observability.NewStaticRuntimeStatusProvider("memory", "memory", true)
	httpObs := observability.NewHTTPObservability(statusProvider)
	healthHandler := observability.NewHealthHandler(statusProvider)
	rateLimiter := security.NewRateLimiter(config.RateLimitConfig{Enabled: false})

	mux := http.NewServeMux()
	healthHandler.RegisterRoutes(mux)
	auditHandler.RegisterRoutes(mux)
	docHandler.RegisterRoutes(mux)
	searchHandler.RegisterRoutes(mux)
	workflowHandler.RegisterRoutes(mux)
	notificationHandler.RegisterRoutes(mux)
	iamAdminHandler.RegisterRoutes(mux)
	mux.Handle("/api/v1/metrics", httpObs.MetricsHandler())

	_ = notifRepo.Create(httptest.NewRequest(http.MethodGet, "/", nil).Context(), notificationdomain.Notification{
		ID:              "notif-contract",
		RecipientUserID: "admin-local",
		EventType:       "workflow.approval.requested",
		ResourceType:    "document",
		ResourceID:      "doc-contract",
		Title:           "Approval requested",
		Message:         "A contract smoke notification is available.",
		Status:          notificationdomain.StatusPending,
		IdempotencyKey:  "notif-contract",
	})

	_ = authRepo.CreateUser(httptest.NewRequest(http.MethodGet, "/", nil).Context(), authdomain.CreateUserParams{
		UserID:             "contract-user",
		Username:           "contract-user",
		DisplayName:        "Contract User",
		PasswordHash:       "hash",
		PasswordAlgo:       "bcrypt",
		MustChangePassword: false,
		IsActive:           true,
		Roles:              []iamdomain.Role{iamdomain.RoleViewer},
		CreatedBy:          "system",
	})

	return httpObs.Wrap(rateLimiter.Wrap(iamMiddleware.Wrap(mux)))
}

func createDocument(t *testing.T, handler http.Handler) string {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"title":           "Marketplace Procedure Seed",
		"documentProfile": "po",
		"processArea":     "marketplaces",
		"ownerId":         "owner-contract",
		"businessUnit":    "commercial",
		"department":      "marketplaces",
		"classification":  "INTERNAL",
		"metadata": map[string]any{
			"procedure_code": "PO-MKT-001",
		},
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

func TestAttachmentSmokeFlow(t *testing.T) {
	handler := buildContractTestHandler()
	docID := createDocument(t, handler)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "contract.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.WriteString(part, "contract attachment"); err != nil {
		t.Fatalf("write attachment: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/"+docID+"/attachments", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-Id", "admin-local")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("upload attachment status mismatch: got=%d body=%s", rr.Code, rr.Body.String())
	}
}
