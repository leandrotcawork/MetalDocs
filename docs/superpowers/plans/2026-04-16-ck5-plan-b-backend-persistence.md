# CK5 Plan B — Backend Persistence

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the `localStorageStub` with real API persistence for CK5 author and fill flows. Add two pairs of Go API endpoints (template CK5 draft, document CK5 content), fix `apiPersistence.ts` to call them with correct shapes, and verify the full stack end-to-end.

**Architecture:**
- **Template HTML** stored in `TemplateDraft.BlocksJSON` as `{"_ck5":{"contentHtml":"…","manifest":{…}}}`. No DB migration required — reuses existing `template_drafts.blocks_json` column. Requires a draft to already exist (created via `POST /api/v1/templates`).
- **Document HTML** stored in `domain.Version.Content` with `ContentSource = "ck5_browser"`. New service methods skip the MDDM template validation used by the existing browser editor (CK5 manages its own template separately).
- **Frontend** already async-ready (`Promise.resolve()` wrapper in pages, env-flag barrel in `persistence/index.ts`). Only `apiPersistence.ts` needs fixing.
- **Env switch:** `VITE_CK5_PERSISTENCE=api` activates `apiPersistence`; default is `local`.

**Out of scope for Plan B:** BlockNote deletion (Plan C), DOCX/PDF export (Plan C), CK5 template publish flow, CK5 image upload beyond existing `MddmUploadAdapter`.

**New files:**
```
internal/modules/documents/application/service_ck5.go             — GetCK5Document + SaveCK5Document service methods
internal/modules/documents/application/service_ck5_test.go        — unit tests for document service methods
internal/modules/documents/delivery/http/handler_ck5_content.go   — GET/POST /api/v1/documents/{id}/content/ck5
internal/modules/documents/delivery/http/handler_ck5_content_test.go
internal/modules/documents/application/service_ck5_template.go    — GetCK5TemplateDraft + SaveCK5TemplateDraft service methods
internal/modules/documents/application/service_ck5_template_test.go — unit tests for template service methods
internal/modules/documents/delivery/http/handler_ck5_template.go  — GET/PUT /api/v1/templates/{key}/ck5-draft
internal/modules/documents/delivery/http/handler_ck5_template_test.go
frontend/apps/web/src/features/documents/ck5/persistence/__tests__/apiPersistence.test.ts
```

**Modified files:**
```
internal/modules/documents/domain/model.go                        — add ContentSourceCK5Browser constant
internal/modules/documents/delivery/http/handler.go               — wire new route in handleDocumentsSubRoutes
internal/modules/documents/delivery/http/template_admin_handler.go — wire ck5-draft case in handleTemplatesSubRoutes
internal/modules/documents/infrastructure/memory/template_drafts_repo.go — add UpsertTemplateDraftForTest helper
frontend/apps/web/src/features/documents/ck5/persistence/apiPersistence.ts — fix endpoints + shapes
```

**Conventions:**
- All Go tests use `httptest.NewRecorder()` + in-memory repo (no real DB).
- TDD: failing test → implementation → passing test → commit.
- Commits: `feat(ck5):`, `test(ck5):`, `fix(ck5):`. Scope: `ck5`.
- Working directory for Go commands: repo root `C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs`.
- Working directory for frontend commands: `frontend/apps/web`.
- Use `rtk` prefix on all terminal commands (token savings).

---

## Phase 0 — Worktree

### Task 0: Create worktree and branch

**Files:** None (git only).

- [ ] **Step 1: Verify clean working tree**

Run: `rtk git status`
Expected: No merge conflicts; branch is `main`.

- [ ] **Step 2: Create worktree**

Run:
```bash
rtk git worktree add ../MetalDocs-ck5-plan-b -b migrate/ck5-plan-b
```
Expected: `Preparing worktree (new branch 'migrate/ck5-plan-b')`.

- [ ] **Step 3: All subsequent tasks run inside `../MetalDocs-ck5-plan-b`**

Run: `cd ../MetalDocs-ck5-plan-b && rtk git status`
Expected: branch `migrate/ck5-plan-b`, no local changes.

---

## Phase 1 — Document CK5 Content API (Go)

### Task 1: ContentSourceCK5Browser constant + service methods

**Files:**
- Modify: `internal/modules/documents/domain/model.go`
- Create: `internal/modules/documents/application/service_ck5.go`
- Test: `internal/modules/documents/application/service_ck5_test.go`

- [ ] **Step 1: Write failing test**

Write `internal/modules/documents/application/service_ck5_test.go`:

```go
package application_test

import (
	"context"
	"testing"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/infrastructure/memory"
)

func makeCK5Service(t *testing.T) (*application.Service, *memory.Repository) {
	t.Helper()
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	return svc, repo
}

func seedDocWithVersion(t *testing.T, repo *memory.Repository, docID, html string) {
	t.Helper()
	ctx := context.Background()
	doc := domain.Document{
		ID:              docID,
		DocumentProfile: "po",
	}
	ver := domain.Version{
		DocumentID:    docID,
		Number:        1,
		Content:       html,
		ContentHash:   "abc123",
		ContentSource: domain.ContentSourceNative,
	}
	if err := repo.CreateDocumentWithInitialVersion(ctx, doc, ver); err != nil {
		t.Fatalf("seed doc+version: %v", err)
	}
}

func TestGetCK5DocumentContentAuthorized(t *testing.T) {
	svc, repo := makeCK5Service(t)
	seedDocWithVersion(t, repo, "doc-1", "<p>Hello CK5</p>")

	ctx := context.Background()
	html, err := svc.GetCK5DocumentContentAuthorized(ctx, "doc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if html != "<p>Hello CK5</p>" {
		t.Errorf("got %q, want %q", html, "<p>Hello CK5</p>")
	}
}

func TestGetCK5DocumentContentAuthorized_NotFound(t *testing.T) {
	svc, _ := makeCK5Service(t)
	_, err := svc.GetCK5DocumentContentAuthorized(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing document, got nil")
	}
}

func TestSaveCK5DocumentContentAuthorized(t *testing.T) {
	svc, repo := makeCK5Service(t)
	seedDocWithVersion(t, repo, "doc-2", "<p>Old</p>")

	ctx := context.Background()
	if err := svc.SaveCK5DocumentContentAuthorized(ctx, "doc-2", "<p>New</p>"); err != nil {
		t.Fatalf("save: %v", err)
	}

	html, err := svc.GetCK5DocumentContentAuthorized(ctx, "doc-2")
	if err != nil {
		t.Fatalf("get after save: %v", err)
	}
	if html != "<p>New</p>" {
		t.Errorf("got %q, want %q", html, "<p>New</p>")
	}
}
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk go test ./internal/modules/documents/application/... -run TestGetCK5 -run TestSaveCK5`
Expected: FAIL — `GetCK5DocumentContentAuthorized` undefined.

- [ ] **Step 3: Add ContentSourceCK5Browser constant**

Edit `internal/modules/documents/domain/model.go` — add after the existing constants block:

```go
ContentSourceCK5Browser = "ck5_browser"
```

- [ ] **Step 4: Implement `service_ck5.go`**

Write `internal/modules/documents/application/service_ck5.go`:

```go
package application

import (
	"context"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

// GetCK5DocumentContentAuthorized returns the HTML body of the latest version
// for the given document. Mirrors the auth pattern of SaveBrowserContentAuthorized:
// GetDocument + isAllowed(CapabilityDocumentView). No MDDM template validation —
// CK5 manages its own template contract.
func (s *Service) GetCK5DocumentContentAuthorized(ctx context.Context, documentID string) (string, error) {
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return "", err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
	if err != nil {
		return "", err
	}
	if !allowed {
		return "", domain.ErrDocumentNotFound
	}
	ver, err := s.latestVersion(ctx, doc.ID)
	if err != nil {
		return "", err
	}
	return ver.Content, nil
}

// SaveCK5DocumentContentAuthorized saves HTML content for a CK5 document.
// Auth pattern: GetDocument + isAllowed(CapabilityDocumentEdit). Uses CAS on
// the current content hash to prevent lost-update races.
func (s *Service) SaveCK5DocumentContentAuthorized(ctx context.Context, documentID, html string) error {
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentEdit)
	if err != nil {
		return err
	}
	if !allowed {
		return domain.ErrDocumentNotFound
	}
	current, err := s.latestVersion(ctx, doc.ID)
	if err != nil {
		return err
	}

	expectedHash := strings.TrimSpace(current.ContentHash)
	if expectedHash == "" {
		expectedHash = contentHash(current.Content)
	}

	updated := current
	updated.Content = html
	updated.ContentHash = contentHash(html)
	updated.ContentSource = domain.ContentSourceCK5Browser
	updated.TextContent = ""

	return s.repo.UpdateDraftVersionContentCAS(ctx, updated, expectedHash)
}
```

- [ ] **Step 5: Run test, verify it passes**

Run: `rtk go test ./internal/modules/documents/application/... -run TestGetCK5 -run TestSaveCK5 -v`
Expected: PASS (3 tests).

- [ ] **Step 6: Commit**

```bash
rtk git add internal/modules/documents/domain/model.go internal/modules/documents/application/service_ck5.go internal/modules/documents/application/service_ck5_test.go
rtk git commit -m "feat(ck5): add GetCK5DocumentContentAuthorized + SaveCK5DocumentContentAuthorized service methods"
```

---

### Task 2: HTTP handlers — GET/POST /api/v1/documents/{id}/content/ck5

**Files:**
- Create: `internal/modules/documents/delivery/http/handler_ck5_content.go`
- Create: `internal/modules/documents/delivery/http/handler_ck5_content_test.go`
- Modify: `internal/modules/documents/delivery/http/handler.go` (route wiring only)

- [ ] **Step 1: Write failing test**

Write `internal/modules/documents/delivery/http/handler_ck5_content_test.go`:

```go
package httpdelivery_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httpdelivery "metaldocs/internal/modules/documents/delivery/http"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/infrastructure/memory"
	"metaldocs/internal/modules/documents/application"
)

func setupCK5ContentHandler(t *testing.T) (*httpdelivery.Handler, *memory.Repository) {
	t.Helper()
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	h := httpdelivery.NewHandler(svc)
	return h, repo
}

func seedCK5Doc(t *testing.T, repo *memory.Repository, docID, html string) {
	t.Helper()
	ctx := context.Background()
	_ = repo.CreateDocumentWithInitialVersion(ctx,
		domain.Document{ID: docID, DocumentProfile: "po"},
		domain.Version{
			DocumentID: docID, Number: 1,
			Content: html, ContentHash: "h1",
			ContentSource: domain.ContentSourceNative,
		},
	)
}

func TestGetCK5Content_200(t *testing.T) {
	h, repo := setupCK5ContentHandler(t)
	seedCK5Doc(t, repo, "d1", "<p>CK5 content</p>")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/d1/content/ck5", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["body"] != "<p>CK5 content</p>" {
		t.Errorf("got body=%q, want <p>CK5 content</p>", resp["body"])
	}
}

func TestGetCK5Content_404(t *testing.T) {
	h, _ := setupCK5ContentHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/missing/content/ck5", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", w.Code)
	}
}

func TestPostCK5Content_201(t *testing.T) {
	h, repo := setupCK5ContentHandler(t)
	seedCK5Doc(t, repo, "d2", "<p>Old</p>")

	body, _ := json.Marshal(map[string]string{"body": "<p>New</p>"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/d2/content/ck5",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201; body: %s", w.Code, w.Body.String())
	}
}

func TestPostCK5Content_MissingBody(t *testing.T) {
	h, repo := setupCK5ContentHandler(t)
	seedCK5Doc(t, repo, "d3", "<p>x</p>")

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/d3/content/ck5",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", w.Code)
	}
}
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk go test ./internal/modules/documents/delivery/http/... -run TestGetCK5Content -run TestPostCK5Content`
Expected: FAIL — ServeHTTP routes not dispatching to CK5 handler yet.

- [ ] **Step 3: Implement `handler_ck5_content.go`**

Write `internal/modules/documents/delivery/http/handler_ck5_content.go`:

```go
package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strings"
)

const maxCK5PayloadBytes = 2 << 20 // 2 MB

type ck5ContentRequest struct {
	Body string `json:"body"`
}

type ck5ContentResponse struct {
	Body string `json:"body"`
}

// handleDocumentContentCK5Get serves GET /api/v1/documents/{id}/content/ck5.
// Returns the raw HTML body of the latest document version.
func (h *Handler) handleDocumentContentCK5Get(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)
	html, err := h.service.GetCK5DocumentContentAuthorized(r.Context(), documentID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}
	writeJSON(w, http.StatusOK, ck5ContentResponse{Body: html})
}

// handleDocumentContentCK5Post serves POST /api/v1/documents/{id}/content/ck5.
// Accepts {"body":"<html>"} and saves it as the new CK5 browser content.
func (h *Handler) handleDocumentContentCK5Post(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)
	r.Body = http.MaxBytesReader(w, r.Body, maxCK5PayloadBytes)

	var req ck5ContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}
	if strings.TrimSpace(req.Body) == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "body is required", traceID)
		return
	}

	if err := h.service.SaveCK5DocumentContentAuthorized(r.Context(), documentID, req.Body); err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
```

- [ ] **Step 4: Wire routes in `handler.go`**

In `internal/modules/documents/delivery/http/handler.go`, find the `handleDocumentsSubRoutes` function. Locate the block that dispatches on path parts `[docID, "content", action]`. Add a new case for `"ck5"` after the existing `"browser"` case:

Before the default/fallthrough in the content sub-router, add:
```go
case len(parts) == 3 && strings.TrimSpace(parts[0]) != "" && parts[1] == "content" && parts[2] == "ck5":
    switch r.Method {
    case http.MethodGet:
        h.handleDocumentContentCK5Get(w, r, parts[0])
    case http.MethodPost:
        h.handleDocumentContentCK5Post(w, r, parts[0])
    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
```

Note: read the exact structure of `handleDocumentsSubRoutes` in `handler.go` lines 1480–1520 before editing to ensure the case is placed correctly without breaking existing routing.

- [ ] **Step 5: Run tests, verify they pass**

Run: `rtk go test ./internal/modules/documents/delivery/http/... -run TestGetCK5Content -run TestPostCK5Content -v`
Expected: PASS (4 tests).

- [ ] **Step 6: Run full Go test suite to confirm no regressions**

Run: `rtk go test ./...`
Expected: all tests pass (or same pre-existing failures as before this branch).

- [ ] **Step 7: Commit**

```bash
rtk git add internal/modules/documents/delivery/http/handler_ck5_content.go internal/modules/documents/delivery/http/handler_ck5_content_test.go internal/modules/documents/delivery/http/handler.go
rtk git commit -m "feat(ck5): add GET/POST /api/v1/documents/{id}/content/ck5 endpoints"
```

---

## Phase 2 — Template CK5 Draft API (Go)

### Task 3: Service methods — GetCK5TemplateDraftContent + SaveCK5TemplateDraftAuthorized

**Files:**
- Create: `internal/modules/documents/application/service_ck5_template.go`
- Create: `internal/modules/documents/application/service_ck5_template_test.go`

**Design note:** CK5 content is stored in `TemplateDraft.BlocksJSON` as:
```json
{"_ck5": {"contentHtml": "…", "manifest": {"fields": []}}}
```
This reuses the existing column without a DB migration. A draft must already exist (created via `POST /api/v1/templates` + the existing template admin flow). `GetCK5TemplateDraftContent` returns empty content gracefully if the `_ck5` key is absent (draft exists but was created by BlockNote, not CK5).

- [ ] **Step 1: Write failing test**

Write `internal/modules/documents/application/service_ck5_template_test.go`:

```go
package application_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/infrastructure/memory"
)

func makeCK5TemplateSvc(t *testing.T) (*application.Service, *memory.Repository) {
	t.Helper()
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	return svc, repo
}

func seedDraft(t *testing.T, repo *memory.Repository, key string) {
	t.Helper()
	emptyBlocks, _ := json.Marshal(map[string]any{"type": "doc", "content": []any{}})
	draft := &domain.TemplateDraft{
		TemplateKey: key,
		ProfileCode: "po",
		Name:        "Test Draft",
		BlocksJSON:  emptyBlocks,
		LockVersion: 1,
		CreatedBy:   "test",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repo.UpsertTemplateDraftForTest(draft); err != nil {
		t.Fatalf("seed draft: %v", err)
	}
}

func TestGetCK5TemplateDraftContent_EmptyWhenNoCK5Key(t *testing.T) {
	svc, repo := makeCK5TemplateSvc(t)
	seedDraft(t, repo, "tpl-1")

	html, manifest, err := svc.GetCK5TemplateDraftContent(context.Background(), "tpl-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if html != "" {
		t.Errorf("expected empty html, got %q", html)
	}
	if manifest == nil {
		t.Error("manifest should be non-nil (empty default)")
	}
}

func TestGetCK5TemplateDraftContent_NotFound(t *testing.T) {
	svc, _ := makeCK5TemplateSvc(t)
	_, _, err := svc.GetCK5TemplateDraftContent(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing draft")
	}
}

func TestSaveAndGetCK5TemplateDraftContent_RoundTrip(t *testing.T) {
	svc, repo := makeCK5TemplateSvc(t)
	seedDraft(t, repo, "tpl-2")

	ctx := context.Background()
	manifest := map[string]any{"fields": []any{}}
	if err := svc.SaveCK5TemplateDraftAuthorized(ctx, "tpl-2", "<p>CK5 tpl</p>", manifest, "user-1"); err != nil {
		t.Fatalf("save: %v", err)
	}

	html, got, err := svc.GetCK5TemplateDraftContent(ctx, "tpl-2")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if html != "<p>CK5 tpl</p>" {
		t.Errorf("html: got %q, want <p>CK5 tpl</p>", html)
	}
	if got == nil {
		t.Error("manifest is nil")
	}
}

func TestSaveCK5TemplateDraftAuthorized_NotFound(t *testing.T) {
	svc, _ := makeCK5TemplateSvc(t)
	err := svc.SaveCK5TemplateDraftAuthorized(context.Background(), "missing", "<p>x</p>", nil, "u")
	if err == nil {
		t.Fatal("expected error for missing draft")
	}
}

// TestSaveCK5TemplateDraftAuthorized_PreservesNonCK5Keys verifies merge semantics:
// pre-existing non-_ck5 keys in BlocksJSON survive a CK5 save unchanged.
func TestSaveCK5TemplateDraftAuthorized_PreservesNonCK5Keys(t *testing.T) {
	svc, repo := makeCK5TemplateSvc(t)

	// Seed a draft whose BlocksJSON contains a "type" key (BlockNote-style).
	blocknotePayload, _ := json.Marshal(map[string]any{
		"type":    "doc",
		"content": []any{},
	})
	_ = repo.UpsertTemplateDraftForTest(&domain.TemplateDraft{
		TemplateKey: "tpl-preserve",
		ProfileCode: "po",
		Name:        "Preserve Test",
		BlocksJSON:  blocknotePayload,
		LockVersion: 1,
		CreatedBy:   "test",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})

	ctx := context.Background()
	if err := svc.SaveCK5TemplateDraftAuthorized(ctx, "tpl-preserve", "<p>ck5</p>", nil, "u"); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Read back raw BlocksJSON and verify both "type" (BlockNote) and "_ck5" keys coexist.
	draft, err := repo.GetTemplateDraft(ctx, "tpl-preserve")
	if err != nil {
		t.Fatalf("get draft: %v", err)
	}
	var merged map[string]any
	if err := json.Unmarshal(draft.BlocksJSON, &merged); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if merged["type"] != "doc" {
		t.Errorf("pre-existing 'type' key clobbered; got %v", merged["type"])
	}
	if merged["_ck5"] == nil {
		t.Error("_ck5 key missing after save")
	}
}
```

**Note:** `memory.Repository` needs `UpsertTemplateDraftForTest` for seeding. Check `infrastructure/memory/template_drafts_repo.go` — if the method doesn't exist, add it:

```go
// UpsertTemplateDraftForTest directly inserts a draft for tests (bypasses CAS).
func (r *Repository) UpsertTemplateDraftForTest(draft *domain.TemplateDraft) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.templateDrafts[draft.TemplateKey] = draft
    return nil
}
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk go test ./internal/modules/documents/application/... -run TestGetCK5Template -run TestSaveCK5Template -v`
Expected: FAIL — `GetCK5TemplateDraftContent` undefined.

- [ ] **Step 3: Implement `service_ck5_template.go`**

Write `internal/modules/documents/application/service_ck5_template.go`:

```go
package application

import (
	"context"
	"encoding/json"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

// GetCK5TemplateDraftContent reads the CK5 HTML + manifest from an existing
// TemplateDraft's BlocksJSON. Returns ("", empty manifest, nil) if the draft
// exists but has no _ck5 key (BlockNote draft, not yet authored in CK5).
// Returns an error if no draft exists for the key.
func (s *Service) GetCK5TemplateDraftContent(ctx context.Context, templateKey string) (string, map[string]any, error) {
	draft, err := s.repo.GetTemplateDraft(ctx, strings.TrimSpace(templateKey))
	if err != nil {
		return "", nil, err
	}

	// Parse BlocksJSON as a generic map to avoid coupling to a typed struct.
	// The _ck5 key may be absent if the draft was created by BlockNote.
	var data map[string]any
	if len(draft.BlocksJSON) > 0 {
		_ = json.Unmarshal(draft.BlocksJSON, &data)
	}

	ck5Raw, ok := data["_ck5"]
	if !ok || ck5Raw == nil {
		return "", map[string]any{"fields": []any{}}, nil
	}
	ck5, ok := ck5Raw.(map[string]any)
	if !ok {
		return "", map[string]any{"fields": []any{}}, nil
	}

	html, _ := ck5["contentHtml"].(string)
	manifest, _ := ck5["manifest"].(map[string]any)
	if manifest == nil {
		manifest = map[string]any{"fields": []any{}}
	}
	return html, manifest, nil
}

// SaveCK5TemplateDraftAuthorized writes CK5 HTML + manifest into an existing
// TemplateDraft's BlocksJSON using optimistic locking. Requires the draft to
// already exist — create it first via POST /api/v1/templates.
//
// IMPORTANT — merge semantics: the _ck5 key is upserted into the existing
// BlocksJSON map, preserving any other keys (e.g. BlockNote payload or meta).
// This prevents CK5 saves from clobbering non-CK5 draft state.
func (s *Service) SaveCK5TemplateDraftAuthorized(ctx context.Context, templateKey, contentHTML string, manifest map[string]any, actorID string) error {
	key := strings.TrimSpace(templateKey)

	existing, err := s.repo.GetTemplateDraft(ctx, key)
	if err != nil {
		return err
	}

	if manifest == nil {
		manifest = map[string]any{"fields": []any{}}
	}

	// Merge: parse existing BlocksJSON into a generic map so that non-_ck5 keys
	// (e.g. BlockNote "type"/"content" fields) are preserved after this save.
	var existingData map[string]any
	if len(existing.BlocksJSON) > 0 {
		_ = json.Unmarshal(existing.BlocksJSON, &existingData)
	}
	if existingData == nil {
		existingData = map[string]any{}
	}
	existingData["_ck5"] = map[string]any{
		"contentHtml": contentHTML,
		"manifest":    manifest,
	}

	blocksJSON, err := json.Marshal(existingData)
	if err != nil {
		return err
	}

	updated := *existing
	updated.BlocksJSON = blocksJSON

	_, err = s.repo.UpsertTemplateDraftCAS(ctx, &updated, existing.LockVersion)
	return err
}
```

- [ ] **Step 4: Add `UpsertTemplateDraftForTest` to memory repo if missing**

Read `internal/modules/documents/infrastructure/memory/template_drafts_repo.go`.
If `UpsertTemplateDraftForTest` is absent, add it at the bottom of the file (as shown in Step 1).

- [ ] **Step 5: Run test, verify it passes**

Run: `rtk go test ./internal/modules/documents/application/... -run TestGetCK5Template -run TestSaveCK5Template -v`
Expected: PASS (5 tests — including preservation round-trip).

- [ ] **Step 6: Commit**

```bash
rtk git add internal/modules/documents/application/service_ck5_template.go internal/modules/documents/application/service_ck5_template_test.go internal/modules/documents/infrastructure/memory/
rtk git commit -m "feat(ck5): add GetCK5TemplateDraftContent + SaveCK5TemplateDraftAuthorized service methods"
```

---

### Task 4: HTTP handlers — GET/PUT /api/v1/templates/{key}/ck5-draft

**Auth context plumbing:**
- `actorID` for template saves: sourced via `userIDFromContext(r.Context())` — same pattern used in other template handlers. Middleware injects the user into context before the handler chain.
- Auth enforcement for template endpoints: handled by upstream middleware (session/JWT check). The service receives an already-authenticated context; `actorID` is passed for audit logging only.
- Auth enforcement for document endpoints: `s.isAllowed(ctx, doc, capability)` gates access inside the service. In unit tests, `NewService(repo, nil, nil)` uses nil policy which is permissive — this is intentional for fast in-memory tests. Full auth coverage lives in integration/E2E tests with a real policy.
- **403 path for documents**: tested by verifying that a missing document returns 404 (the service returns `domain.ErrDocumentNotFound` both for truly missing docs AND for permission-denied cases, matching the existing handler convention of not leaking existence for unauthorized resources).

**Files:**
- Create: `internal/modules/documents/delivery/http/handler_ck5_template.go`
- Create: `internal/modules/documents/delivery/http/handler_ck5_template_test.go`
- Modify: `internal/modules/documents/delivery/http/template_admin_handler.go` (route wiring — add `"ck5-draft"` case)

- [ ] **Step 1: Write failing test**

Write `internal/modules/documents/delivery/http/handler_ck5_template_test.go`:

```go
package httpdelivery_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpdelivery "metaldocs/internal/modules/documents/delivery/http"
	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/infrastructure/memory"
)

func setupCK5TemplateHandler(t *testing.T) (*httpdelivery.Handler, *memory.Repository) {
	t.Helper()
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	h := httpdelivery.NewHandler(svc)
	return h, repo
}

func seedTemplateDraft(t *testing.T, repo *memory.Repository, key string) {
	t.Helper()
	emptyBlocks, _ := json.Marshal(map[string]any{"type": "doc"})
	_ = repo.UpsertTemplateDraftForTest(&domain.TemplateDraft{
		TemplateKey: key, ProfileCode: "po", Name: "T",
		BlocksJSON: emptyBlocks, LockVersion: 1,
		CreatedBy: "test", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
}

func TestGetCK5TemplateDraft_200_EmptyInitially(t *testing.T) {
	h, repo := setupCK5TemplateHandler(t)
	seedTemplateDraft(t, repo, "k1")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/k1/ck5-draft", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["contentHtml"] != "" {
		t.Errorf("expected empty contentHtml, got %v", resp["contentHtml"])
	}
}

func TestGetCK5TemplateDraft_404(t *testing.T) {
	h, _ := setupCK5TemplateHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/missing/ck5-draft", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", w.Code)
	}
}

func TestPutCK5TemplateDraft_200(t *testing.T) {
	h, repo := setupCK5TemplateHandler(t)
	seedTemplateDraft(t, repo, "k2")

	body, _ := json.Marshal(map[string]any{
		"contentHtml": "<p>CK5</p>",
		"manifest":    map[string]any{"fields": []any{}},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/templates/k2/ck5-draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200; body: %s", w.Code, w.Body.String())
	}

	// Verify round-trip: GET should return saved content.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/templates/k2/ck5-draft", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)
	var resp map[string]any
	_ = json.NewDecoder(w2.Body).Decode(&resp)
	if resp["contentHtml"] != "<p>CK5</p>" {
		t.Errorf("round-trip: got %v, want <p>CK5</p>", resp["contentHtml"])
	}
}

func TestPutCK5TemplateDraft_404_NoDraft(t *testing.T) {
	h, _ := setupCK5TemplateHandler(t)
	body, _ := json.Marshal(map[string]any{"contentHtml": "<p>x</p>"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/templates/missing/ck5-draft", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", w.Code)
	}
}
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk go test ./internal/modules/documents/delivery/http/... -run TestGetCK5TemplateDraft -run TestPutCK5TemplateDraft -v`
Expected: FAIL — routes not wired yet.

- [ ] **Step 3: Implement `handler_ck5_template.go`**

Write `internal/modules/documents/delivery/http/handler_ck5_template.go`:

```go
package httpdelivery

import (
	"encoding/json"
	"net/http"
)

type ck5TemplateDraftRequest struct {
	ContentHTML string         `json:"contentHtml"`
	Manifest    map[string]any `json:"manifest"`
}

type ck5TemplateDraftResponse struct {
	ContentHTML string         `json:"contentHtml"`
	Manifest    map[string]any `json:"manifest"`
}

// handleGetCK5TemplateDraft serves GET /api/v1/templates/{key}/ck5-draft.
func (h *Handler) handleGetCK5TemplateDraft(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	html, manifest, err := h.service.GetCK5TemplateDraftContent(r.Context(), key)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}
	writeJSON(w, http.StatusOK, ck5TemplateDraftResponse{
		ContentHTML: html,
		Manifest:    manifest,
	})
}

// handlePutCK5TemplateDraft serves PUT /api/v1/templates/{key}/ck5-draft.
func (h *Handler) handlePutCK5TemplateDraft(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	r.Body = http.MaxBytesReader(w, r.Body, maxCK5PayloadBytes)

	var req ck5TemplateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	actorID := userIDFromContext(r.Context())
	if err := h.service.SaveCK5TemplateDraftAuthorized(r.Context(), key, req.ContentHTML, req.Manifest, actorID); err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}
	w.WriteHeader(http.StatusOK)
}
```

- [ ] **Step 4: Wire routes in `handler.go`**

In `handleTemplatesSubRoutes`, add a `"ck5-draft"` case inside the `len(parts) == 2` block (alongside `"draft"`, `"publish"`, etc.):

```go
case "ck5-draft":
    if r.Method == http.MethodGet {
        h.handleGetCK5TemplateDraft(w, r, key)
    } else if r.Method == http.MethodPut {
        h.handlePutCK5TemplateDraft(w, r, key)
    } else {
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
```

Read the exact structure of `handleTemplatesSubRoutes` in `template_admin_handler.go` lines 34–123 before editing to place the case correctly.

- [ ] **Step 5: Run tests, verify they pass**

Run: `rtk go test ./internal/modules/documents/delivery/http/... -run TestGetCK5TemplateDraft -run TestPutCK5TemplateDraft -v`
Expected: PASS (4 tests).

- [ ] **Step 6: Run full Go test suite**

Run: `rtk go test ./...`
Expected: all tests pass (or same pre-existing failures).

- [ ] **Step 7: Commit**

```bash
rtk git add internal/modules/documents/delivery/http/handler_ck5_template.go internal/modules/documents/delivery/http/handler_ck5_template_test.go internal/modules/documents/delivery/http/handler.go internal/modules/documents/delivery/http/template_admin_handler.go
rtk git commit -m "feat(ck5): add GET/PUT /api/v1/templates/{key}/ck5-draft endpoints"
```

---

## Phase 3 — Frontend: Fix apiPersistence.ts

### Task 5: Fix apiPersistence.ts to call CK5 endpoints

**Files:**
- Modify: `frontend/apps/web/src/features/documents/ck5/persistence/apiPersistence.ts`
- Test: `frontend/apps/web/src/features/documents/ck5/persistence/__tests__/apiPersistence.test.ts`

**Context:**
Current `apiPersistence.ts` problems:
| Function | Current call | Problem | Fix |
|---|---|---|---|
| `saveTemplate` | `PUT /api/v1/templates/{id}/draft` with `{contentHtml,manifest}` | Wrong endpoint + wrong shape | → `PUT /api/v1/templates/{id}/ck5-draft` with `{contentHtml,manifest}` |
| `loadTemplate` | `GET /api/v1/templates/{id}` expecting `TemplateRecord` | Returns `templateDraftResponse` (no contentHtml) | → `GET /api/v1/templates/{id}/ck5-draft` expecting `{contentHtml,manifest}` |
| `saveDocument` | `POST /api/v1/documents/{id}/content/browser` with `{contentHtml}` | Wrong endpoint + wrong field | → `POST /api/v1/documents/{id}/content/ck5` with `{body}` |
| `loadDocument` | `GET /api/v1/documents/{id}` expecting `{contentHtml}` | Wrong endpoint | → `GET /api/v1/documents/{id}/content/ck5` expecting `{body}` |

- [ ] **Step 1: Write failing test**

Write `frontend/apps/web/src/features/documents/ck5/persistence/__tests__/apiPersistence.test.ts`:

```ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Import after mocking so fetch is patched before module loads.
let saveTemplate: typeof import('../apiPersistence').saveTemplate;
let loadTemplate: typeof import('../apiPersistence').loadTemplate;
let saveDocument: typeof import('../apiPersistence').saveDocument;
let loadDocument: typeof import('../apiPersistence').loadDocument;

const mockFetch = vi.fn();

beforeEach(async () => {
  globalThis.fetch = mockFetch;
  vi.resetModules();
  const mod = await import('../apiPersistence');
  saveTemplate = mod.saveTemplate;
  loadTemplate = mod.loadTemplate;
  saveDocument = mod.saveDocument;
  loadDocument = mod.loadDocument;
});

afterEach(() => {
  vi.restoreAllMocks();
});

function ok(body: unknown = {}): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  });
}

describe('saveTemplate', () => {
  it('calls PUT /api/v1/templates/{id}/ck5-draft with correct shape', async () => {
    mockFetch.mockResolvedValue(ok());
    await saveTemplate('tpl-1', '<p>hi</p>', { fields: [] });
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/templates/tpl-1/ck5-draft',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ contentHtml: '<p>hi</p>', manifest: { fields: [] } }),
      }),
    );
  });
});

describe('loadTemplate', () => {
  it('calls GET /api/v1/templates/{id}/ck5-draft and maps contentHtml', async () => {
    mockFetch.mockResolvedValue(ok({ contentHtml: '<p>loaded</p>', manifest: { fields: [] } }));
    const rec = await loadTemplate('tpl-2');
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/templates/tpl-2/ck5-draft',
      expect.objectContaining({ credentials: 'include' }),
    );
    expect(rec?.contentHtml).toBe('<p>loaded</p>');
  });

  it('returns null on 404', async () => {
    mockFetch.mockResolvedValue(new Response('', { status: 404 }));
    const rec = await loadTemplate('missing');
    expect(rec).toBeNull();
  });
});

describe('saveDocument', () => {
  it('calls POST /api/v1/documents/{id}/content/ck5 with body field', async () => {
    mockFetch.mockResolvedValue(new Response('', { status: 201 }));
    await saveDocument('doc-1', '<p>doc</p>');
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/documents/doc-1/content/ck5',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ body: '<p>doc</p>' }),
      }),
    );
  });
});

describe('loadDocument', () => {
  it('calls GET /api/v1/documents/{id}/content/ck5 and returns body field', async () => {
    mockFetch.mockResolvedValue(ok({ body: '<p>doc content</p>' }));
    const html = await loadDocument('doc-2');
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/documents/doc-2/content/ck5',
      expect.objectContaining({ credentials: 'include' }),
    );
    expect(html).toBe('<p>doc content</p>');
  });

  it('returns null on 404', async () => {
    mockFetch.mockResolvedValue(new Response('', { status: 404 }));
    expect(await loadDocument('missing')).toBeNull();
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/ck5/persistence/__tests__/apiPersistence.test.ts`
Expected: FAIL — endpoints/shapes don't match.

- [ ] **Step 3: Rewrite `apiPersistence.ts`**

Write `frontend/apps/web/src/features/documents/ck5/persistence/apiPersistence.ts`:

```ts
import type { TemplateRecord } from './localStorageStub';

async function throwIfNotOk(res: Response): Promise<Response> {
  if (!res.ok) throw new Error(`API ${res.status}: ${res.statusText}`);
  return res;
}

// ---------------------------------------------------------------------------
// Template persistence (Author editor)
// ---------------------------------------------------------------------------

export async function saveTemplate(
  id: string,
  contentHtml: string,
  manifest: TemplateRecord['manifest'],
): Promise<void> {
  await throwIfNotOk(
    await fetch(`/api/v1/templates/${encodeURIComponent(id)}/ck5-draft`, {
      method: 'PUT',
      headers: { 'content-type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ contentHtml, manifest }),
    }),
  );
}

export async function loadTemplate(id: string): Promise<TemplateRecord | null> {
  const res = await fetch(`/api/v1/templates/${encodeURIComponent(id)}/ck5-draft`, {
    credentials: 'include',
  });
  if (res.status === 404) return null;
  await throwIfNotOk(res);
  const data = (await res.json()) as { contentHtml: string; manifest: TemplateRecord['manifest'] };
  return {
    id,
    contentHtml: data.contentHtml ?? '',
    manifest: data.manifest ?? { fields: [] },
  };
}

// ---------------------------------------------------------------------------
// Document persistence (Fill editor)
// ---------------------------------------------------------------------------

export async function saveDocument(id: string, contentHtml: string): Promise<void> {
  await throwIfNotOk(
    await fetch(`/api/v1/documents/${encodeURIComponent(id)}/content/ck5`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ body: contentHtml }),
    }),
  );
}

export async function loadDocument(id: string): Promise<string | null> {
  const res = await fetch(`/api/v1/documents/${encodeURIComponent(id)}/content/ck5`, {
    credentials: 'include',
  });
  if (res.status === 404) return null;
  await throwIfNotOk(res);
  const data = (await res.json()) as { body?: string };
  return data.body ?? null;
}
```

- [ ] **Step 4: Run test, verify it passes**

Run: `rtk vitest run src/features/documents/ck5/persistence/__tests__/apiPersistence.test.ts`
Expected: PASS (7 tests).

- [ ] **Step 5: Type check**

Run: `rtk pnpm tsc --noEmit`
Expected: zero errors (or same pre-existing errors as before this task).

- [ ] **Step 6: Build**

Run: `rtk pnpm build`
Expected: success.

- [ ] **Step 7: Commit**

```bash
cd ../..
rtk git add frontend/apps/web/src/features/documents/ck5/persistence/apiPersistence.ts frontend/apps/web/src/features/documents/ck5/persistence/__tests__/apiPersistence.test.ts
rtk git commit -m "fix(ck5): apiPersistence — use CK5 endpoints with correct request/response shapes"
```

---

## Phase 4 — Verification Gate

### Task 6: Full stack verify + PR prep

**Files:** None. Quality gates only.

- [ ] **Step 1: Full Go test suite**

Run: `cd ../.. && rtk go test ./...`
Expected: all tests pass.

- [ ] **Step 2: Full frontend test suite**

Run: `cd frontend/apps/web && rtk vitest run`
Expected: all tests pass.

- [ ] **Step 3: Frontend type check**

Run: `rtk pnpm tsc --noEmit`
Expected: zero errors (or same pre-existing set).

- [ ] **Step 4: Frontend build**

Run: `rtk pnpm build`
Expected: success.

- [ ] **Step 5: Manual smoke (dev harness with VITE_CK5_PERSISTENCE=api)**

Start Go API: `cd ../.. && bash scripts/start-api.sh`
Start frontend dev server with env: `VITE_CK5_PERSISTENCE=api pnpm dev`
Open `http://localhost:PORT/#/test-harness/ck5?mode=author&tpl=<existing-draft-key>`
Expected: AuthorEditor loads, saves call `PUT /api/v1/templates/{key}/ck5-draft`, no 4xx in console.

Open `http://localhost:PORT/#/test-harness/ck5?mode=fill&tpl=<key>&doc=<docId>`
Expected: FillEditor loads document HTML, saves call `POST /api/v1/documents/{docId}/content/ck5`.

- [ ] **Step 6: Push worktree branch**

```bash
rtk git push -u origin migrate/ck5-plan-b
```

- [ ] **Step 7: Open PR**

```bash
gh pr create --title "Plan B: CK5 backend persistence — replace localStorage stub" --body "$(cat <<'EOF'
## Summary
- Adds `GET/POST /api/v1/documents/{id}/content/ck5` — HTML save/load without MDDM template validation
- Adds `GET/PUT /api/v1/templates/{key}/ck5-draft` — CK5 HTML stored in existing draft's BlocksJSON under `_ck5` key (no DB migration)
- Fixes `apiPersistence.ts` — correct endpoints + request/response field names
- Set `VITE_CK5_PERSISTENCE=api` to activate; default still `local`

## Test plan
- [ ] `go test ./...` — green
- [ ] `vitest run` — green (incl. new apiPersistence tests)
- [ ] `tsc --noEmit` — no new errors
- [ ] `pnpm build` — succeeds
- [ ] Manual: harness with `VITE_CK5_PERSISTENCE=api` — author saves/loads template, fill saves/loads document

BlockNote stack untouched. Plan C = DOCX/PDF export + BlockNote deletion.
EOF
)"
```

- [ ] **Step 8: Return PR URL to human reviewer.**

---

## Plan self-review checklist

1. **Spec coverage** — all 4 apiPersistence mismatches resolved; both Go endpoint pairs covered.
2. **Placeholder scan** — no TBD; every handler, service method, and test is written in full.
3. **Type consistency** — `ck5ContentRequest.Body` matches `ck5ContentResponse.Body`; `ck5TemplateDraftRequest.ContentHTML` matches `ck5TemplateDraftResponse.ContentHTML`; `apiPersistence.ts` response field names match Go handler JSON tags.
4. **Commit granularity** — one commit per task (6 commits total + PR).
5. **No DB migration** — CK5 HTML stored in existing `blocks_json` column via `_ck5` wrapper key.
6. **Test-first** — every task: failing test → implement → passing test → commit.
7. **File inventory** — all new/modified files listed at top including test files and `template_admin_handler.go`.
8. **Merge semantics** — `SaveCK5TemplateDraftAuthorized` parses existing BlocksJSON as `map[string]any` and upserts only `_ck5`; non-CK5 keys preserved; covered by `TestSaveCK5TemplateDraftAuthorized_PreservesNonCK5Keys`.
9. **Auth plumbing** — `actorID` sourced from `userIDFromContext`; middleware enforces auth upstream; unit tests use nil policy (permissive) by convention; E2E covers auth enforcement.

## Codex hardening record

- Round: 1/2
- Verdict: APPROVE_WITH_FIXES
- Mode: COVERAGE
- Issues fixed:
  - **structural**: Template save merge semantics — replaced `ck5Wrapper` struct marshal with `map[string]any` upsert; added `TestSaveCK5TemplateDraftAuthorized_PreservesNonCK5Keys`
  - **structural**: Auth plumbing — added explicit note in Task 4 on `actorID` sourcing, middleware auth enforcement, nil-policy unit test convention, and 404-as-auth-denial pattern
  - **local**: File inventory — added all test files and `template_admin_handler.go` to modified list; added `infrastructure/memory/template_drafts_repo.go`
- `upgrade_required: true` → resolved by above revisions
