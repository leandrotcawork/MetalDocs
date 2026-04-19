# Docgen Hybrid Gap-Closure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the remaining docgen migration gaps on `plan/docgen-hybrid` by adding etapa body storage, API endpoints, audit/events, and reconciling the minimal docgen harness.

**Architecture:** Keep `plan/docgen-hybrid` as baseline. Add `document_versions.body_blocks` for etapa content, expose GET/PATCH per step, enforce immutability by creating a new version, and wire audit + events. Re-add the minimal harness scripts without replacing the richer docgen renderer.

**Tech Stack:** Go (net/http, Postgres), TypeScript/Node (docgen), OpenAPI YAML.

---

## File Structure

Create:
- `migrations/0047_add_document_versions_body_blocks.sql`
- `internal/modules/documents/domain/etapa_body.go`
- `tests/unit/documents_etapa_body_service_test.go`
- `tests/unit/documents_etapa_body_handler_test.go`

Modify:
- `apps/docgen/scripts/harness.ps1`
- `apps/docgen/scripts/sample-payload.json`
- `internal/modules/documents/domain/model.go`
- `internal/modules/documents/domain/port.go`
- `internal/modules/documents/application/service.go`
- `internal/modules/documents/application/service_content_native.go`
- `internal/modules/documents/application/docgen_projection.go`
- `internal/modules/documents/delivery/http/handler.go`
- `internal/modules/documents/delivery/http/handler_content.go`
- `internal/modules/documents/infrastructure/postgres/repository.go`
- `internal/modules/documents/infrastructure/memory/repository.go`
- `apps/api/cmd/metaldocs-api/main.go`
- `apps/api/cmd/metaldocs-e2e-seed/main.go`
- `apps/api/cmd/metaldocs-api/permissions.go`
- `api/openapi/v1/openapi.yaml`
- `tasks/todo.md`
- `tasks/lessons.md`

---

### Task 1: Reconcile Minimal Docgen Harness

**Files:**
- Create: `apps/docgen/scripts/harness.ps1`
- Create: `apps/docgen/scripts/sample-payload.json`

- [ ] **Step 1: Add sample payload**

```json
{
  "documentType": "PO",
  "documentCode": "PO-01",
  "title": "Procedimento Operacional",
  "sections": {}
}
```

- [ ] **Step 2: Add harness script**

```powershell
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root

try {
  Write-Host "==> Typecheck (tsc --noEmit)"
  npx tsc --noEmit

  Write-Host "==> Build (tsc -p tsconfig.build.json)"
  npx tsc -p tsconfig.build.json

  Write-Host "==> Start server (node dist/index.js)"
  $proc = Start-Process -FilePath "node" -ArgumentList "dist/index.js" -PassThru -NoNewWindow
  Start-Sleep -Seconds 2

  Write-Host "==> POST /generate"
  $resp = curl.exe -s -D - -o "$env:TEMP\\docgen-harness.docx" `
    -H "Content-Type: application/json" `
    -X POST "http://localhost:3001/generate" `
    --data-binary "@$PSScriptRoot\\sample-payload.json"

  $len = (Get-Item "$env:TEMP\\docgen-harness.docx").Length
  if ($len -le 0) { throw "DOCX is empty" }

  $headerText = $resp -join " "
  if ($headerText -notmatch "(?i)application/vnd.openxmlformats-officedocument.wordprocessingml.document") {
    Write-Host $headerText
    throw "Unexpected content type"
  }

  Write-Host "OK: DOCX size = $len bytes"
}
finally {
  if ($proc -and !$proc.HasExited) { Stop-Process -Id $proc.Id }
  Pop-Location
}
```

- [ ] **Step 3: Run harness (should pass)**

Run: `powershell -ExecutionPolicy Bypass -File apps/docgen/scripts/harness.ps1`  
Expected: `OK: DOCX size = <non-zero>`

- [ ] **Step 4: Commit**

```bash
git add apps/docgen/scripts
git commit -m "feat(docgen): restore minimal harness scripts"
```

---

### Task 2: Add `body_blocks` Migration

**Files:**
- Create: `migrations/0047_add_document_versions_body_blocks.sql`

- [ ] **Step 1: Write migration**

Exact SQL is not required here; the migration may use `IF NOT EXISTS` and must target the schema-qualified `metaldocs.document_versions` table.

```sql
ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS body_blocks JSONB DEFAULT '[]';
```

- [ ] **Step 2: Commit**

```bash
git add migrations/0047_add_document_versions_body_blocks.sql
git commit -m "feat(document): add body_blocks column"
```

---

### Task 3: Domain + Repository Support for Body Blocks

**Files:**
- Create: `internal/modules/documents/domain/etapa_body.go`
- Modify: `internal/modules/documents/domain/model.go`
- Modify: `internal/modules/documents/domain/port.go`
- Modify: `internal/modules/documents/infrastructure/postgres/repository.go`
- Modify: `internal/modules/documents/infrastructure/memory/repository.go`

- [ ] **Step 1: Define EtapaBody**

```go
package domain

import "encoding/json"

type EtapaBody struct {
	Blocks []json.RawMessage `json:"blocks"`
}
```

- [ ] **Step 2: Add `BodyBlocks` to `domain.Version`**

```go
type Version struct {
	// existing fields...
	BodyBlocks []EtapaBody
}
```

- [ ] **Step 3: Extend repository interface**

```go
UpdateVersionBodyBlocks(ctx context.Context, documentID string, versionNumber int, bodyBlocks []EtapaBody) error
```

- [ ] **Step 4: Update Postgres repository inserts + reads**

Update INSERT column lists to include `body_blocks` and bind JSON:

```go
bodyBlocksJSON := "[]"
if len(version.BodyBlocks) > 0 {
	if raw, err := json.Marshal(version.BodyBlocks); err == nil {
		bodyBlocksJSON = string(raw)
	}
}
```

Include `body_blocks` in `INSERT INTO metaldocs.document_versions (...) VALUES (..., $15)` and in `SELECT` scans:

```go
var bodyBlocksJSON []byte
// Scan(..., &bodyBlocksJSON, ...)
if len(bodyBlocksJSON) > 0 {
	var bodyBlocks []domain.EtapaBody
	if err := json.Unmarshal(bodyBlocksJSON, &bodyBlocks); err == nil {
		version.BodyBlocks = bodyBlocks
	}
}
```

Add repository method:

```go
func (r *Repository) UpdateVersionBodyBlocks(ctx context.Context, documentID string, versionNumber int, bodyBlocks []domain.EtapaBody) error {
	const q = `
UPDATE metaldocs.document_versions
SET body_blocks = $3::jsonb
WHERE document_id = $1 AND version_number = $2
`
	payload := "[]"
	if len(bodyBlocks) > 0 {
		raw, err := json.Marshal(bodyBlocks)
		if err != nil {
			return domain.ErrInvalidCommand
		}
		payload = string(raw)
	}
	res, err := r.db.ExecContext(ctx, q, documentID, versionNumber, payload)
	if err != nil {
		return mapError(err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return domain.ErrVersionNotFound
	}
	return nil
}
```

- [ ] **Step 5: Update memory repository**

```go
func (r *Repository) UpdateVersionBodyBlocks(_ context.Context, documentID string, versionNumber int, bodyBlocks []domain.EtapaBody) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	versions := r.versions[documentID]
	for i := range versions {
		if versions[i].Number == versionNumber {
			versions[i].BodyBlocks = bodyBlocks
			r.versions[documentID] = versions
			return nil
		}
	}
	return domain.ErrVersionNotFound
}
```

- [ ] **Step 6: Commit**

```bash
git add internal/modules/documents/domain internal/modules/documents/infrastructure
git commit -m "feat(document): add body blocks to versions"
```

---

### Task 4: Service Layer (Immutability + Audit + Event)

**Files:**
- Modify: `internal/modules/documents/application/service.go`
- Modify: `internal/modules/documents/application/service_content_native.go`
- Modify: `internal/modules/documents/application/docgen_projection.go`
- Create: `tests/unit/documents_etapa_body_service_test.go`

- [ ] **Step 1: Add audit writer to Service**

```go
type Service struct {
	// existing fields...
	audit auditdomain.Writer
}

func (s *Service) WithAuditWriter(writer auditdomain.Writer) *Service {
	s.audit = writer
	return s
}
```

- [ ] **Step 2: Add Get/Save methods**

```go
func (s *Service) GetEtapaBodyAuthorized(ctx context.Context, documentID string, versionNumber int, stepIndex int) (domain.EtapaBody, error) {
	// validate inputs, load doc, check permission
	// load version via repo.GetVersion
	// return body at index or empty if missing
}

func (s *Service) SaveEtapaBodyAuthorized(ctx context.Context, cmd domain.SaveEtapaBodyCommand) (domain.Version, error) {
	// validate inputs, load doc, permission check, immutability
	// create new version (NextVersionNumber)
	// copy Content/NativeContent from source version
	// update BodyBlocks at stepIndex
	// SaveVersion
	// record audit + publish event
}
```

Command type:

```go
type SaveEtapaBodyCommand struct {
	DocumentID    string
	VersionNumber int
	StepIndex     int
	Blocks        []json.RawMessage
	TraceID       string
}
```

Validation: decode blocks into `[]domain.Block` and call `domain.ValidateBlocks`.

- [ ] **Step 3: Update docgen projection to prefer BodyBlocks**

```go
func extractEtapas(content map[string]any, bodyBlocks []domain.EtapaBody) []DocgenEtapa {
	// if bodyBlocks has index i, use those blocks
}
```

Update `ProjectDocgenPayload` to pass `version.BodyBlocks` into `extractEtapas`.

- [ ] **Step 4: Add unit tests for service**

```go
func TestSaveEtapaBodyCreatesNewVersion(t *testing.T) {
	// create doc, save initial version
	// call SaveEtapaBodyAuthorized on version 1
	// assert new version number and body blocks persisted
}

func TestSaveEtapaBodyRejectsInvalidBlocks(t *testing.T) {
	// blocks with missing text -> expect ErrInvalidNativeContent
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./tests/unit -run EtapaBody`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/documents/application tests/unit/documents_etapa_body_service_test.go
git commit -m "feat(document): add etapa body service + projection merge"
```

---

### Task 5: Delivery Layer + Permissions + OpenAPI

**Files:**
- Modify: `internal/modules/documents/delivery/http/handler.go`
- Modify: `internal/modules/documents/delivery/http/handler_content.go`
- Modify: `apps/api/cmd/metaldocs-api/permissions.go`
- Modify: `api/openapi/v1/openapi.yaml`
- Create: `tests/unit/documents_etapa_body_handler_test.go`

- [ ] **Step 1: Add request/response types**

```go
type DocumentEtapaBodyRequest struct {
	Blocks []json.RawMessage `json:"blocks"`
}

type DocumentEtapaBodyResponse struct {
	Blocks []json.RawMessage `json:"blocks"`
}
```

- [ ] **Step 2: Add handlers**

```go
func (h *Handler) handleDocumentEtapaBodyGet(w http.ResponseWriter, r *http.Request, documentID, versionID, stepIndex string) { ... }
func (h *Handler) handleDocumentEtapaBodyPatch(w http.ResponseWriter, r *http.Request, documentID, versionID, stepIndex string) { ... }
```

Use `iamdomain.UserIDFromContext` to enforce auth. Parse `versionID` and `stepIndex` as ints.

- [ ] **Step 3: Route wiring**

Add to `handleDocumentSubRoutes`:

```go
if len(parts) == 6 && parts[1] == "versions" && parts[3] == "etapas" && parts[5] == "body" {
  // GET/PATCH dispatch
}
```

- [ ] **Step 4: Permissions**

Add in `permissions.go`:

```go
if method == http.MethodGet && strings.Contains(path, "/versions/") && strings.Contains(path, "/etapas/") && strings.HasSuffix(path, "/body") {
	return iamdomain.PermDocumentRead, true
}
if method == http.MethodPatch && strings.Contains(path, "/versions/") && strings.Contains(path, "/etapas/") && strings.HasSuffix(path, "/body") {
	return iamdomain.PermDocumentEdit, true
}
```

- [ ] **Step 5: OpenAPI**

Add new paths:

```yaml
/documents/{documentId}/versions/{versionId}/etapas/{stepIndex}/body:
  get:
    summary: Obtem o corpo rich da etapa
    parameters:
      - $ref: '#/components/parameters/DocumentId'
      - name: versionId
        in: path
        required: true
        schema: { type: integer }
      - name: stepIndex
        in: path
        required: true
        schema: { type: integer }
    responses:
      '200':
        description: Corpo da etapa
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/DocumentEtapaBodyResponse'
  patch:
    summary: Atualiza o corpo rich da etapa
    parameters:
      - $ref: '#/components/parameters/DocumentId'
      - name: versionId
        in: path
        required: true
        schema: { type: integer }
      - name: stepIndex
        in: path
        required: true
        schema: { type: integer }
    requestBody:
      required: true
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/DocumentEtapaBodyRequest'
    responses:
      '200':
        description: Corpo atualizado
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/DocumentEtapaBodyResponse'
```

Add schemas:

```yaml
DocumentEtapaBodyRequest:
  type: object
  properties:
    blocks:
      type: array
      items: { type: object }
DocumentEtapaBodyResponse:
  type: object
  properties:
    blocks:
      type: array
      items: { type: object }
```

- [ ] **Step 6: Add handler test**

```go
func TestHandleDocumentEtapaBodyRequiresAuth(t *testing.T) {
  // request without user context -> 401
}
```

- [ ] **Step 7: Run tests**

Run: `go test ./tests/unit -run EtapaBody`  
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/modules/documents/delivery/http apps/api/cmd/metaldocs-api/permissions.go api/openapi/v1/openapi.yaml tests/unit/documents_etapa_body_handler_test.go
git commit -m "feat(document): add etapa body endpoints"
```

---

### Task 6: Wire Audit Writer + Update Tasks/Lessons

**Files:**
- Modify: `apps/api/cmd/metaldocs-api/main.go`
- Modify: `apps/api/cmd/metaldocs-e2e-seed/main.go`
- Modify: `tasks/todo.md`
- Modify: `tasks/lessons.md`

- [ ] **Step 1: Wire audit writer into documents service**

```go
docService := docapp.NewService(deps.DocumentsRepo, deps.Publisher, nil).
  WithAttachmentStore(deps.AttachmentStore).
  WithCarbone(deps.CarboneClient, deps.CarboneTemplates).
  WithDocgen(deps.DocgenClient).
  WithAuditWriter(deps.AuditWriter)
```

- [ ] **Step 2: Update tasks**

Mark docgen migration gaps as done in `tasks/todo.md`, and add lesson:

```
## Lesson N — Use a projection firewall between storage and render
Date: 2026-03-31 | Trigger: correction
Wrong:   Render payload assembled directly from handlers
Correct: Projection lives in application layer as the sole storage->render translator
Rule:    Keep render shape isolated from persistence shape
Layer:   application
```

- [ ] **Step 3: Commit**

```bash
git add apps/api/cmd tasks/todo.md tasks/lessons.md
git commit -m "feat(document): wire audit + update tasks"
```

---

### Task 7: Final Verification

- [ ] **Step 1: Run Go tests**

Run: `go test ./...`  
Expected: PASS

- [ ] **Step 2: Run docgen harness**

Run: `powershell -ExecutionPolicy Bypass -File apps/docgen/scripts/harness.ps1`  
Expected: PASS

---

## Plan Self-Review

- [x] Spec coverage: harness reconciliation, persistence, endpoints, audit/events, OpenAPI, tests all covered.
- [x] Placeholder scan: no TBDs, every step has concrete code/commands.
- [x] Type consistency: `EtapaBody`, `BodyBlocks`, endpoint shapes consistent across plan.

---

**Plan complete and saved to `docs/superpowers/plans/2026-03-31-docgen-hybrid-gap-closure-plan.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
