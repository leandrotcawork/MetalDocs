//go:build integration

package httpdelivery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	docapp "metaldocs/internal/modules/documents/application"
	documentmemory "metaldocs/internal/modules/documents/infrastructure/memory"
	iamdomain "metaldocs/internal/modules/iam/domain"
	"metaldocs/internal/platform/config"
	"metaldocs/internal/platform/render/docgen"
	workflowapp "metaldocs/internal/modules/workflow/application"
	workflowdelivery "metaldocs/internal/modules/workflow/delivery/http"
	workflowmemory "metaldocs/internal/modules/workflow/infrastructure/memory"
)

func TestAPI_DocumentMatrix_CreateSaveConflictReleaseExport(t *testing.T) {
	docgenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("docgen method = %s, want %s", r.Method, http.MethodPost)
		}
		switch r.URL.Path {
		case "/generate", "/generate-browser":
			w.Header().Set("Content-Type", docxContentType)
			_, _ = w.Write([]byte("docx-integration"))
		default:
			t.Fatalf("docgen path = %s", r.URL.Path)
		}
	}))
	defer docgenServer.Close()

	repo := documentmemory.NewRepository()
	attachments := documentmemory.NewAttachmentStore()
	docService := docapp.NewService(repo, nil, nil).
		WithAttachmentStore(attachments).
		WithDocgenClient(docgen.NewClient(config.DocgenConfig{
			Enabled:               true,
			APIURL:                docgenServer.URL,
			RequestTimeoutSeconds: 1,
		}))
	docHandler := NewHandler(docService)

	workflowService := workflowapp.NewService(repo, workflowmemory.NewApprovalRepository(), nil, nil, nil)
	workflowHandler := workflowdelivery.NewHandler(workflowService)

	mux := http.NewServeMux()
	docHandler.RegisterRoutes(mux)
	workflowHandler.RegisterRoutes(mux)

	var created DocumentCreatedResponse

	t.Run("create", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/documents", strings.NewReader(`{
			"title":"Integration Matrix Document",
			"documentType":"po",
			"documentProfile":"po",
			"ownerId":"owner-1",
			"businessUnit":"operations",
			"department":"sgq",
			"classification":"INTERNAL"
		}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("decode create response: %v", err)
		}
		if strings.TrimSpace(created.DocumentID) == "" {
			t.Fatalf("documentId is empty; body=%s", rec.Body.String())
		}
		if created.Status != "DRAFT" {
			t.Fatalf("status = %q, want %q", created.Status, "DRAFT")
		}
	})

	t.Run("save conflict", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/"+created.DocumentID+"/content/native", strings.NewReader(`{
			"draftToken":"v1:stale",
			"content":{"field":"value"}
		}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusConflict, rec.Body.String())
		}

		var envelope apiErrorEnvelope
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("decode error envelope: %v", err)
		}
		if envelope.Error.Code != "DRAFT_CONFLICT" {
			t.Fatalf("code = %q, want %q", envelope.Error.Code, "DRAFT_CONFLICT")
		}
	})

	t.Run("release shape via workflow transition", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/documents/"+created.DocumentID+"/transitions", strings.NewReader(`{
			"toStatus":"IN_REVIEW",
			"reason":"integration matrix release request",
			"assignedReviewer":"reviewer-1"
		}`))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "owner-1", nil))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var payload struct {
			DocumentID string `json:"documentId"`
			ToStatus   string `json:"toStatus"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode release response: %v", err)
		}
		if payload.DocumentID != created.DocumentID {
			t.Fatalf("documentId = %q, want %q", payload.DocumentID, created.DocumentID)
		}
		if payload.ToStatus != "IN_REVIEW" {
			t.Fatalf("toStatus = %q, want %q", payload.ToStatus, "IN_REVIEW")
		}
	})

	t.Run("export", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/"+created.DocumentID+"/export/docx", nil)
		req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "owner-1", nil))
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}
		if got := rec.Header().Get("Content-Type"); got != docxContentType {
			t.Fatalf("content-type = %q, want %q", got, docxContentType)
		}
		if body := rec.Body.String(); body != "docx-integration" {
			t.Fatalf("body = %q, want %q", body, "docx-integration")
		}
	})
}
