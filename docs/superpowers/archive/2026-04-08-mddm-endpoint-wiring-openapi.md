# MDDM Endpoint Wiring + OpenAPI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire Task 59/60 handlers into live API routes with permission enforcement and OpenAPI v1 contract coverage.

**Architecture:** Keep the existing documents HTTP handler as the route hub, but inject dedicated MDDM load/submit handlers via composition. Build adapters at API composition time to bridge the MDDM postgres repository types into application services without violating package boundaries. Update permission resolver and OpenAPI in the same slice.

**Tech Stack:** Go 1.25, net/http ServeMux, PostgreSQL repository adapters, OpenAPI 3.0 YAML

---

## File Structure

- Modify: `internal/platform/bootstrap/api.go`
  - Add an MDDM repository dependency (`*postgres.MDDMRepository`) in `APIDependencies` for postgres mode.
- Modify: `apps/api/cmd/metaldocs-api/main.go`
  - Compose `LoadService` and `SubmitForApprovalService` using adapter types and pass into documents handler.
- Modify: `internal/modules/documents/delivery/http/handler.go`
  - Add optional MDDM handler fields + `WithMDDMHandlers(...)`.
  - Wire `/documents/{documentId}/load` and `/documents/{documentId}/submit-for-approval`.
- Modify: `apps/api/cmd/metaldocs-api/permissions.go`
  - Add permission mapping for new routes.
- Modify: `api/openapi/v1/openapi.yaml`
  - Add paths + response schema for load and submit-for-approval.
- Add: `apps/api/cmd/metaldocs-api/mddm_adapters.go`
  - Adapter that maps `postgres.DocumentVersion` to `application.LoadVersion`.
- Add: `internal/modules/documents/delivery/http/handler_mddm_wiring_test.go`
  - Route-level wiring tests for new endpoints through `Handler.handleDocumentSubRoutes`.

---

### Task 1: Bootstrap + Main Wiring

**Files:**
- Modify: `internal/platform/bootstrap/api.go`
- Modify: `apps/api/cmd/metaldocs-api/main.go`
- Create: `apps/api/cmd/metaldocs-api/mddm_adapters.go`

- [ ] **Step 1: Add MDDM repo dependency in bootstrap**

```go
// internal/platform/bootstrap/api.go (imports)
pgrepo "metaldocs/internal/modules/documents/infrastructure/postgres"

type APIDependencies struct {
	// ...
	MDDMRepo *pgrepo.MDDMRepository
}

// postgres branch in BuildAPIDependencies:
return APIDependencies{
	DocumentsRepo: pgrepo.NewRepository(db),
	MDDMRepo:      pgrepo.NewMDDMRepository(db),
	// ...
}, nil
```

- [ ] **Step 2: Create adapter for load service**

```go
// apps/api/cmd/metaldocs-api/mddm_adapters.go
package main

import (
	"context"
	"encoding/json"

	docapp "metaldocs/internal/modules/documents/application"
	pgrepo "metaldocs/internal/modules/documents/infrastructure/postgres"
)

type mddmLoadRepoAdapter struct {
	repo *pgrepo.MDDMRepository
}

func (a *mddmLoadRepoAdapter) GetActiveDraft(ctx context.Context, documentID, userID string) (*docapp.LoadVersion, error) {
	if a == nil || a.repo == nil {
		return nil, nil
	}
	row, err := a.repo.GetActiveDraftForUser(ctx, documentID, userID)
	if err != nil || row == nil {
		return nil, err
	}
	return &docapp.LoadVersion{
		DocumentID:      row.DocumentID,
		Version:         row.VersionNumber,
		Status:          row.Status,
		Content:         json.RawMessage(row.ContentBlocks),
		TemplateKey:     readTemplateKey(row.TemplateRef),
		TemplateVersion: readTemplateVersion(row.TemplateRef),
		ContentHash:     row.ContentHash,
	}, nil
}

func (a *mddmLoadRepoAdapter) GetCurrentReleased(ctx context.Context, documentID string) (*docapp.LoadVersion, error) {
	if a == nil || a.repo == nil {
		return nil, nil
	}
	row, err := a.repo.GetCurrentReleased(ctx, documentID)
	if err != nil || row == nil {
		return nil, err
	}
	return &docapp.LoadVersion{
		DocumentID:      row.DocumentID,
		Version:         row.VersionNumber,
		Status:          row.Status,
		Content:         json.RawMessage(row.ContentBlocks),
		TemplateKey:     readTemplateKey(row.TemplateRef),
		TemplateVersion: readTemplateVersion(row.TemplateRef),
		ContentHash:     row.ContentHash,
	}, nil
}
```

- [ ] **Step 3: Wire MDDM services into doc handler in main**

```go
// apps/api/cmd/metaldocs-api/main.go
var loadSvc *docapp.LoadService
var submitSvc *docapp.SubmitForApprovalService
if deps.MDDMRepo != nil {
	loadSvc = docapp.NewLoadService(&mddmLoadRepoAdapter{repo: deps.MDDMRepo})
	submitSvc = docapp.NewSubmitForApprovalService(deps.MDDMRepo)
}

docHandler := docdelivery.NewHandler(docService).
	WithAttachmentDownloads(...).
	WithMDDMHandlers(loadSvc, submitSvc)
```

- [ ] **Step 4: Run targeted compile check**

Run: `go test ./apps/api/cmd/metaldocs-api/... -run TestNonExistent -count=1`
Expected: `ok` or `[no test files]` with successful compile.

- [ ] **Step 5: Commit**

```bash
git add internal/platform/bootstrap/api.go apps/api/cmd/metaldocs-api/main.go apps/api/cmd/metaldocs-api/mddm_adapters.go
git commit -m "feat(mddm): compose load/submit services in api wiring"
```

---

### Task 2: Route Wiring In Documents Handler

**Files:**
- Modify: `internal/modules/documents/delivery/http/handler.go`
- Create: `internal/modules/documents/delivery/http/handler_mddm_wiring_test.go`

- [ ] **Step 1: Add injectable handlers + setter**

```go
type Handler struct {
	service               *application.Service
	signer                *security.AttachmentSigner
	downloadTTL           time.Duration
	loadHandler           *LoadHandler
	submitForApprovalHandler *SubmitForApprovalHandler
}

func (h *Handler) WithMDDMHandlers(load *application.LoadService, submit *application.SubmitForApprovalService) *Handler {
	if load != nil {
		h.loadHandler = NewLoadHandler(load)
	}
	if submit != nil {
		h.submitForApprovalHandler = NewSubmitForApprovalHandler(submit)
	}
	return h
}
```

- [ ] **Step 2: Wire `/load` and `/submit-for-approval` in subroutes**

```go
if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && parts[1] == "load" && r.Method == http.MethodGet {
	if h.loadHandler == nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Load handler is not configured", requestTraceID(r))
		return
	}
	h.loadHandler.Load(w, r)
	return
}
if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && parts[1] == "submit-for-approval" && r.Method == http.MethodPost {
	if h.submitForApprovalHandler == nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Submit-for-approval handler is not configured", requestTraceID(r))
		return
	}
	h.submitForApprovalHandler.SubmitForApproval(w, r)
	return
}
```

- [ ] **Step 3: Add route-level wiring tests**

```go
func TestHandleDocumentSubRoutes_LoadRouteUsesMDDMHandler(t *testing.T) {
	// Create Handler with WithMDDMHandlers(loadSvc, nil), call /api/v1/documents/PO-118/load
	// assert != 404 route not found.
}

func TestHandleDocumentSubRoutes_SubmitForApprovalRouteUsesMDDMHandler(t *testing.T) {
	// Create Handler with WithMDDMHandlers(nil, submitSvc), call /api/v1/documents/PO-118/submit-for-approval?draft_id=<uuid>
	// assert != 404 route not found.
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/modules/documents/delivery/http/... -run "TestHandleDocumentSubRoutes_(LoadRouteUsesMDDMHandler|SubmitForApprovalRouteUsesMDDMHandler|LoadHandler)"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/delivery/http/handler.go internal/modules/documents/delivery/http/handler_mddm_wiring_test.go
git commit -m "feat(mddm): wire load and submit-for-approval routes in documents handler"
```

---

### Task 3: Permission Resolver Mapping

**Files:**
- Modify: `apps/api/cmd/metaldocs-api/permissions.go`

- [ ] **Step 1: Add permission rules for new routes**

```go
if method == http.MethodGet && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/load") {
	return iamdomain.PermDocumentRead, true
}
if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/submit-for-approval") {
	return iamdomain.PermWorkflowTransition, true
}
```

- [ ] **Step 2: Run permissions package tests**

Run: `go test ./apps/api/cmd/metaldocs-api/... -run TestNonExistent -count=1`
Expected: compile success.

- [ ] **Step 3: Commit**

```bash
git add apps/api/cmd/metaldocs-api/permissions.go
git commit -m "feat(authz): map permissions for mddm load and submit-for-approval routes"
```

---

### Task 4: OpenAPI Contract Update

**Files:**
- Modify: `api/openapi/v1/openapi.yaml`

- [ ] **Step 1: Add `/documents/{documentId}/load` path**

```yaml
/documents/{documentId}/load:
  get:
    summary: Carrega o bundle MDDM para edicao (draft do usuario ou released fallback)
    parameters:
      - name: documentId
        in: path
        required: true
        schema:
          type: string
    responses:
      '200':
        description: Conteudo MDDM carregado
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/DocumentMDDMLoadResponse'
      '400':
        description: Requisicao invalida
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ApiErrorEnvelope'
      '401':
        description: Nao autenticado
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ApiErrorEnvelope'
      '404':
        description: Documento nao encontrado
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ApiErrorEnvelope'
```

- [ ] **Step 2: Add `/documents/{documentId}/submit-for-approval` path**

```yaml
/documents/{documentId}/submit-for-approval:
  post:
    summary: Transiciona draft MDDM para pending_approval
    parameters:
      - name: documentId
        in: path
        required: true
        schema:
          type: string
      - name: draft_id
        in: query
        required: true
        schema:
          type: string
          format: uuid
    responses:
      '200':
        description: Draft submetido para aprovacao
      '400':
        description: Requisicao invalida
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ApiErrorEnvelope'
      '401':
        description: Nao autenticado
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ApiErrorEnvelope'
      '422':
        description: Draft em estado invalido para transicao
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ApiErrorEnvelope'
```

- [ ] **Step 3: Add response schema**

```yaml
DocumentMDDMLoadTemplate:
  type: object
  required: [key, version]
  properties:
    key:
      type: string
    version:
      type: integer
      minimum: 0
DocumentMDDMLoadResponse:
  type: object
  required: [documentId, version, status, content, template, contentHash]
  properties:
    documentId:
      type: string
    version:
      type: integer
      minimum: 1
    status:
      type: string
      enum: [draft, pending_approval, released]
    content:
      type: object
      additionalProperties: true
    template:
      $ref: '#/components/schemas/DocumentMDDMLoadTemplate'
    contentHash:
      type: string
```

- [ ] **Step 4: Validate YAML parses**

Run: `go test ./... -run TestNonExistent -count=1`
Expected: compile passes (contract file is syntactically valid and does not break build tooling).

- [ ] **Step 5: Commit**

```bash
git add api/openapi/v1/openapi.yaml
git commit -m "docs(api): add mddm load and submit-for-approval endpoints"
```

---

### Task 5: End-to-End Verification For This Slice

**Files:**
- Modify (if needed): `internal/modules/documents/delivery/http/handler_mddm_wiring_test.go`

- [ ] **Step 1: Run focused backend tests**

Run:
```bash
$env:GOCACHE = (Join-Path (Get-Location) '.gocache')
go test ./internal/modules/documents/application/... -run "TestLoadService|TestSubmitForApprovalService"
go test ./internal/modules/documents/delivery/http/... -run "TestLoadHandler|TestHandleDocumentSubRoutes_LoadRouteUsesMDDMHandler|TestHandleDocumentSubRoutes_SubmitForApprovalRouteUsesMDDMHandler|TestReleaseHandler"
go test ./apps/api/cmd/metaldocs-api/... -run TestNonExistent -count=1
```
Expected: all PASS / compile success.

- [ ] **Step 2: Commit any final test-only fixes**

```bash
git add internal/modules/documents/delivery/http/handler_mddm_wiring_test.go
git commit -m "test(mddm): verify wired routes and permission-safe method guards"
```

- [ ] **Step 3: Final status check**

Run: `git status --short --branch`
Expected: clean working tree on `feature/mddm-foundational`.

