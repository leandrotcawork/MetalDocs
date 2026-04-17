package httpdelivery

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"metaldocs/internal/modules/documents/application"
	documentmemory "metaldocs/internal/modules/documents/infrastructure/memory"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

// newTemplateTestHandler creates a Handler wired to a fresh in-memory repository.
func newTemplateTestHandler(t *testing.T) (*Handler, *documentmemory.Repository) {
	t.Helper()
	repo := documentmemory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	return NewHandler(svc), repo
}

// ctxAdmin returns a context with admin role (has template.edit + template.publish + template.export).
func ctxAdmin() *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	return req.WithContext(iamdomain.WithAuthContext(req.Context(), "actor-admin", []iamdomain.Role{iamdomain.RoleAdmin}))
}

func withAdminCtx(req *http.Request) *http.Request {
	return req.WithContext(iamdomain.WithAuthContext(req.Context(), "actor-admin", []iamdomain.Role{iamdomain.RoleAdmin}))
}

// ---------------------------------------------------------------------------
// POST /api/v1/templates — Create
// ---------------------------------------------------------------------------

func TestHandleCreateTemplate_201(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	body := `{"profileCode":"po","name":"My Template"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", strings.NewReader(body))
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleCreateTemplate(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}

	var resp templateDraftResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.TemplateKey == "" {
		t.Error("expected non-empty templateKey in response")
	}
	if resp.ProfileCode != "po" {
		t.Errorf("profileCode = %q, want po", resp.ProfileCode)
	}
	if resp.Name != "My Template" {
		t.Errorf("name = %q, want My Template", resp.Name)
	}
	if resp.Status != "draft" {
		t.Errorf("status = %q, want draft", resp.Status)
	}
}

func TestHandleCreateTemplate_401_NoUser(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	body := `{"profileCode":"po","name":"My Template"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", strings.NewReader(body))
	// No auth context
	rec := httptest.NewRecorder()

	h.handleCreateTemplate(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /api/v1/templates?profileCode=X — List
// ---------------------------------------------------------------------------

func TestHandleListTemplates_200(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	// First create a draft so there's something to list (using collection handler).
	createBody := `{"profileCode":"po","name":"Template A"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/templates", strings.NewReader(createBody))
	createReq = withAdminCtx(createReq)
	createRec := httptest.NewRecorder()
	h.handleCreateTemplate(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create setup failed: %d %s", createRec.Code, createRec.Body.String())
	}

	// Then publish it so it appears in the versions list.
	var draftResp templateDraftResponse
	json.NewDecoder(bytes.NewReader(createRec.Body.Bytes())).Decode(&draftResp)
	key := draftResp.TemplateKey
	lockVersion := draftResp.LockVersion

	publishBody, _ := json.Marshal(map[string]int{"lockVersion": lockVersion})
	publishReq := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+key+"/publish", bytes.NewReader(publishBody))
	publishReq = withAdminCtx(publishReq)
	publishRec := httptest.NewRecorder()
	h.handlePublish(publishRec, publishReq, key)
	if publishRec.Code != http.StatusOK {
		t.Fatalf("publish setup failed: %d %s", publishRec.Code, publishRec.Body.String())
	}

	// Now list templates for profile "po".
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/templates?profileCode=po", nil)
	listReq = withAdminCtx(listReq)
	listRec := httptest.NewRecorder()
	h.handleListTemplates(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", listRec.Code, listRec.Body.String())
	}

	var resp struct {
		Items []templateVersionResponse `json:"items"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) == 0 {
		t.Error("expected at least one item in list")
	}
}

func TestHandleListTemplates_400_MissingProfileCode(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates", nil)
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleListTemplates(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// PUT /api/v1/templates/{key}/draft — SaveDraft
// ---------------------------------------------------------------------------

func TestHandleSaveDraft_200(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	// Create draft first.
	draft := createTestDraft(t, h, "po", "Save Test")
	key := draft.TemplateKey

	body, _ := json.Marshal(saveDraftRequest{
		Blocks:      json.RawMessage(`[{"type":"paragraph"}]`),
		LockVersion: draft.LockVersion,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/templates/"+key+"/draft", bytes.NewReader(body))
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleSaveDraft(rec, req, key)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var resp templateDraftResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.LockVersion <= draft.LockVersion {
		t.Errorf("lockVersion should have incremented; got %d, had %d", resp.LockVersion, draft.LockVersion)
	}
}

func TestHandleSaveDraft_409_LockConflict(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	draft := createTestDraft(t, h, "po", "Lock Test")
	key := draft.TemplateKey

	// Send with wrong lock version.
	body, _ := json.Marshal(saveDraftRequest{
		Blocks:      json.RawMessage(`[]`),
		LockVersion: 999,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/templates/"+key+"/draft", bytes.NewReader(body))
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleSaveDraft(rec, req, key)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// POST /api/v1/templates/{key}/publish — Publish
// ---------------------------------------------------------------------------

func TestHandlePublish_200(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	draft := createTestDraft(t, h, "po", "Publish Test")
	key := draft.TemplateKey

	body, _ := json.Marshal(publishRequest{LockVersion: draft.LockVersion})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+key+"/publish", bytes.NewReader(body))
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handlePublish(rec, req, key)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var resp templateVersionResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Version != 1 {
		t.Errorf("version = %d, want 1", resp.Version)
	}
}

func TestHandlePublish_422_HasStrippedFields(t *testing.T) {
	h, repo := newTemplateTestHandler(t)

	draft := createTestDraft(t, h, "po", "Stripped Test")
	key := draft.TemplateKey

	// Manually set HasStrippedFields on the draft in repo.
	storedDraft, _ := repo.GetTemplateDraft(ctxAdmin().Context(), key)
	storedDraft.HasStrippedFields = true
	repo.UpsertTemplateDraftCAS(ctxAdmin().Context(), storedDraft, storedDraft.LockVersion)

	// Re-fetch to get current lockVersion.
	freshDraft, _ := repo.GetTemplateDraft(ctxAdmin().Context(), key)

	body, _ := json.Marshal(publishRequest{LockVersion: freshDraft.LockVersion})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+key+"/publish", bytes.NewReader(body))
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handlePublish(rec, req, key)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlePublish_422_ReturnsStructuredValidationErrors(t *testing.T) {
	t.Skip("MDDM block-type validation removed in Plan C — CK5 publish uses ApproveTemplate path")
	// original test preserved below for reference
	h, _ := newTemplateTestHandler(t)

	draft := createTestDraft(t, h, "po", "Invalid Publish Test")
	key := draft.TemplateKey

	invalidBlocks := json.RawMessage(`[{"id":"b1","type":"UNKNOWN_BLOCK_TYPE","props":{},"content":[],"children":[]}]`)
	saved, err := h.service.SaveDraftAuthorized(ctxAdmin().Context(), key, invalidBlocks, nil, nil, draft.LockVersion, "actor-admin")
	if err != nil {
		t.Fatalf("SaveDraftAuthorized() error = %v", err)
	}

	body, _ := json.Marshal(publishRequest{LockVersion: saved.LockVersion})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+key+"/publish", bytes.NewReader(body))
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handlePublish(rec, req, key)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", rec.Code, rec.Body.String())
	}

	var resp publishValidationErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Errors) == 0 {
		t.Fatalf("expected structured publish errors, got none: %+v", resp)
	}
	if resp.Error.Code != "TEMPLATE_PUBLISH_VALIDATION" {
		t.Fatalf("error code = %q, want TEMPLATE_PUBLISH_VALIDATION", resp.Error.Code)
	}
}

// ---------------------------------------------------------------------------
// POST /api/v1/templates/{key}/discard-draft — DiscardDraft
// ---------------------------------------------------------------------------

func TestHandleDiscardDraft_204(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	draft := createTestDraft(t, h, "po", "Discard Test")
	key := draft.TemplateKey

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+key+"/discard-draft", nil)
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleDiscardDraft(rec, req, key)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// 401 for unauthenticated requests
// ---------------------------------------------------------------------------

func TestHandleTemplatesCollection_401_NoUser(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates?profileCode=po", nil)
	rec := httptest.NewRecorder()

	h.handleTemplatesCollection(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// 404 for draft not found
// ---------------------------------------------------------------------------

func TestHandleSaveDraft_404_DraftNotFound(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	body, _ := json.Marshal(saveDraftRequest{
		Blocks:      json.RawMessage(`[]`),
		LockVersion: 1,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/templates/no-such-key/draft", bytes.NewReader(body))
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleSaveDraft(rec, req, "no-such-key")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Sub-route dispatcher
// ---------------------------------------------------------------------------

func TestHandleTemplatesSubRoutes_MethodNotAllowed(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/templates/some-key/publish", nil)
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleTemplatesSubRoutes(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestHandleTemplatesSubRoutes_ImportPost(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	importData := `{"name":"Imported","profileCode":"po","definition":{"type":"page","children":[]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/import?profileCode=po", strings.NewReader(importData))
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleTemplatesSubRoutes(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Clone
// ---------------------------------------------------------------------------

func TestHandleClone_201(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	draft := createTestDraft(t, h, "po", "Source Template")
	key := draft.TemplateKey

	body, _ := json.Marshal(cloneRequest{NewName: "Cloned Template"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+key+"/clone", bytes.NewReader(body))
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleClone(rec, req, key)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}

	var resp templateDraftResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Name != "Cloned Template" {
		t.Errorf("name = %q, want Cloned Template", resp.Name)
	}
	if resp.TemplateKey == key {
		t.Error("clone should have a different key than source")
	}
}

// ---------------------------------------------------------------------------
// Export + Preview DOCX
// ---------------------------------------------------------------------------

func TestHandleExportTemplate_200(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	draft := createTestDraft(t, h, "po", "Export Test")
	key := draft.TemplateKey

	publishBody, _ := json.Marshal(publishRequest{LockVersion: draft.LockVersion})
	publishReq := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+key+"/publish", bytes.NewReader(publishBody))
	publishReq = withAdminCtx(publishReq)
	publishRec := httptest.NewRecorder()
	h.handlePublish(publishRec, publishReq, key)
	if publishRec.Code != http.StatusOK {
		t.Fatalf("publish setup failed: %d %s", publishRec.Code, publishRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/"+key+"/export?version=1", nil)
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleExportTemplate(rec, req, key)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want %q", got, "application/json")
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "attachment") {
		t.Fatalf("content-disposition = %q, want it to contain %q", got, "attachment")
	}
	if !json.Valid(rec.Body.Bytes()) {
		t.Fatalf("export body is not valid json: %s", rec.Body.String())
	}

	var exported map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &exported); err != nil {
		t.Fatalf("decode export body: %v", err)
	}
	if got, _ := exported["templateKey"].(string); got != key {
		t.Fatalf("templateKey = %q, want %q", got, key)
	}
	if got, _ := exported["version"].(float64); int(got) != 1 {
		t.Fatalf("version = %v, want 1", exported["version"])
	}
	if _, ok := exported["definition"]; !ok {
		t.Fatalf("expected export to include definition field")
	}
}

func TestHandleTemplatePreviewDocx_503_RenderUnavailable(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	draft := createTestDraft(t, h, "po", "Preview Test")
	key := draft.TemplateKey

	saveBody, _ := json.Marshal(saveDraftRequest{
		Blocks:      json.RawMessage(`{"blocks":[{"id":"p1","type":"paragraph","children":[{"text":"Preview body"}]}]}`),
		LockVersion: draft.LockVersion,
	})
	saveReq := httptest.NewRequest(http.MethodPut, "/api/v1/templates/"+key+"/draft", bytes.NewReader(saveBody))
	saveReq = withAdminCtx(saveReq)
	saveRec := httptest.NewRecorder()
	h.handleSaveDraft(saveRec, saveReq, key)
	if saveRec.Code != http.StatusOK {
		t.Fatalf("save setup failed: %d %s", saveRec.Code, saveRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+key+"/preview-docx", nil)
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleTemplatePreviewDocx(rec, req, key)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body: %s", rec.Code, rec.Body.String())
	}

	var envelope apiErrorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envelope.Error.Code != "RENDER_UNAVAILABLE" {
		t.Fatalf("error.code = %q, want %q", envelope.Error.Code, "RENDER_UNAVAILABLE")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// createTestDraft creates a draft via the handler and returns the response.
func createTestDraft(t *testing.T, h *Handler, profileCode, name string) templateDraftResponse {
	t.Helper()

	body, _ := json.Marshal(createTemplateRequest{ProfileCode: profileCode, Name: name})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", bytes.NewReader(body))
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleCreateTemplate(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("createTestDraft: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp templateDraftResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("createTestDraft: decode: %v", err)
	}
	return resp
}
