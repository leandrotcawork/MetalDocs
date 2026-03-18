package unit

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/application"
	httpdelivery "metaldocs/internal/modules/documents/delivery/http"
	"metaldocs/internal/modules/documents/infrastructure/memory"
	"metaldocs/internal/platform/observability"
	"metaldocs/internal/platform/security"
)

func newTestMux() *http.ServeMux {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, nil).WithAttachmentStore(memory.NewAttachmentStore())
	h := httpdelivery.NewHandler(svc).
		WithAttachmentDownloads(security.NewAttachmentSigner("test-attachment-secret"), 5*time.Minute)
	mux := http.NewServeMux()
	observability.NewHealthHandler(observability.NewStaticRuntimeStatusProvider("memory", "memory", true)).RegisterRoutes(mux)
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
		var payload map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("invalid health json: %v", err)
		}
		if payload["status"] == nil {
			t.Fatalf("expected health status in payload for %s", path)
		}
	}
}

func TestCreateAndListVersionsFlow(t *testing.T) {
	mux := newTestMux()

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/documents", strings.NewReader(`{"title":"Procedimento de Marketplaces","documentProfile":"po","processArea":"marketplaces","ownerId":"u1","businessUnit":"commercial","department":"marketplaces","classification":"INTERNAL","metadata":{"procedure_code":"PO-MKT-002"}}`))
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
	if created["documentProfile"] != "po" {
		t.Fatalf("expected documentProfile po, got %v", created["documentProfile"])
	}
	if created["documentFamily"] != "procedure" {
		t.Fatalf("expected documentFamily procedure, got %v", created["documentFamily"])
	}
	if created["processArea"] != "marketplaces" {
		t.Fatalf("expected processArea marketplaces, got %v", created["processArea"])
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

func TestListDocumentFamiliesAndProfiles(t *testing.T) {
	mux := newTestMux()

	for _, path := range []string{"/api/v1/document-families", "/api/v1/document-profiles", "/api/v1/process-areas", "/api/v1/document-subjects", "/api/v1/document-profiles/it/schema", "/api/v1/document-profiles/it/governance"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", path, rr.Code)
		}
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

func TestUploadListAndDownloadAttachmentFlow(t *testing.T) {
	mux := newTestMux()

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/documents", strings.NewReader(`{"title":"Instrucao","documentType":"it","ownerId":"u1","businessUnit":"ops","department":"general","metadata":{"instruction_code":"IT-HTTP"}}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	mux.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createRR.Code, createRR.Body.String())
	}

	var created struct {
		DocumentID string `json:"documentId"`
	}
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "manual.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("hello attachment")); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	uploadReq := httptest.NewRequest(http.MethodPost, "/api/v1/documents/"+created.DocumentID+"/attachments", &body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("X-User-Id", "editor-local")
	uploadRR := httptest.NewRecorder()
	mux.ServeHTTP(uploadRR, uploadReq)
	if uploadRR.Code != http.StatusCreated {
		t.Fatalf("expected 201 for upload, got %d body=%s", uploadRR.Code, uploadRR.Body.String())
	}

	var attachment struct {
		AttachmentID string `json:"attachmentId"`
	}
	if err := json.Unmarshal(uploadRR.Body.Bytes(), &attachment); err != nil {
		t.Fatalf("decode upload response: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+created.DocumentID+"/attachments", nil)
	listRR := httptest.NewRecorder()
	mux.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for list attachments, got %d body=%s", listRR.Code, listRR.Body.String())
	}

	urlReq := httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+created.DocumentID+"/attachments/"+attachment.AttachmentID+"/download-url", nil)
	urlRR := httptest.NewRecorder()
	mux.ServeHTTP(urlRR, urlReq)
	if urlRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for download url, got %d body=%s", urlRR.Code, urlRR.Body.String())
	}

	var downloadResp struct {
		DownloadURL string `json:"downloadUrl"`
	}
	if err := json.Unmarshal(urlRR.Body.Bytes(), &downloadResp); err != nil {
		t.Fatalf("decode download url response: %v", err)
	}

	downloadReq := httptest.NewRequest(http.MethodGet, downloadResp.DownloadURL, nil)
	downloadRR := httptest.NewRecorder()
	mux.ServeHTTP(downloadRR, downloadReq)
	if downloadRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for attachment download, got %d body=%s", downloadRR.Code, downloadRR.Body.String())
	}
	if strings.TrimSpace(downloadRR.Body.String()) != "hello attachment" {
		t.Fatalf("unexpected attachment content: %s", downloadRR.Body.String())
	}
}
