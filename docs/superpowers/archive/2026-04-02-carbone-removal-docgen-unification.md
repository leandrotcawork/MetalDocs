# Carbone Removal & Docgen Pipeline Unification — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove all Carbone dependencies, unify the save/preview/export flows into a single docgen pipeline, and add Gotenberg for DOCX→PDF conversion.

**Architecture:** All three document rendering paths (save, render-pdf, export) share a single `buildDocgenPayload()` function → `docgenClient.Generate()` → DOCX bytes. PDF conversion goes through Gotenberg (HTTP sidecar). Carbone is deleted entirely.

**Tech Stack:** Go 1.24, PostgreSQL 16, TypeScript/Node (docx-js), Docker (Gotenberg 8)

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/platform/render/gotenberg/client.go` | Create | Gotenberg HTTP client for DOCX→PDF conversion |
| `internal/platform/config/gotenberg.go` | Create | Gotenberg config from env vars |
| `internal/modules/documents/application/service.go` | Modify | Remove Carbone fields/imports, consolidate docgen client |
| `internal/modules/documents/application/service_content_native.go` | Modify | Rewrite save/render-pdf to use unified docgen pipeline |
| `internal/modules/documents/application/service_content_docx.go` | Modify | Replace Carbone in convertDocxToPDF with Gotenberg |
| `internal/modules/documents/application/service_document_runtime.go` | Modify | Extract shared `buildDocgenPayload()` function |
| `internal/modules/documents/delivery/http/handler_content.go` | Modify | Stub or remove Carbone-only template handlers |
| `internal/modules/documents/delivery/http/handler.go` | Modify | Remove profile template docx route |
| `internal/platform/bootstrap/api.go` | Modify | Remove Carbone deps, add Gotenberg |
| `apps/api/cmd/metaldocs-api/main.go` | Modify | Remove Carbone wiring, add Gotenberg wiring |
| `internal/platform/render/carbone/` | Delete | Entire package |
| `internal/platform/config/carbone.go` | Delete | Carbone config |
| `deploy/compose/docker-compose.yml` | Modify | Remove Carbone service, add Gotenberg service |

---

### Task 1: Create Gotenberg client and config

**Files:**
- Create: `internal/platform/config/gotenberg.go`
- Create: `internal/platform/render/gotenberg/client.go`

- [ ] **Step 1: Create Gotenberg config**

```go
// internal/platform/config/gotenberg.go
package config

import "os"

type GotenbergConfig struct {
	Enabled bool
	URL     string
}

func LoadGotenbergConfig() GotenbergConfig {
	url := os.Getenv("METALDOCS_GOTENBERG_URL")
	if url == "" {
		return GotenbergConfig{}
	}
	return GotenbergConfig{
		Enabled: true,
		URL:     url,
	}
}
```

- [ ] **Step 2: Create Gotenberg client**

```go
// internal/platform/render/gotenberg/client.go
package gotenberg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) ConvertDocxToPDF(ctx context.Context, docxContent []byte) ([]byte, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("files", "document.docx")
	if err != nil {
		return nil, fmt.Errorf("gotenberg: create form file: %w", err)
	}
	if _, err := part.Write(docxContent); err != nil {
		return nil, fmt.Errorf("gotenberg: write docx content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("gotenberg: close multipart: %w", err)
	}

	url := c.baseURL + "/forms/libreoffice/convert"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return nil, fmt.Errorf("gotenberg: create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gotenberg: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gotenberg: status %d: %s", resp.StatusCode, string(respBody))
	}

	return io.ReadAll(resp.Body)
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd internal && go build ./...`
Expected: No errors (new package, no consumers yet).

- [ ] **Step 4: Commit**

```bash
git add internal/platform/config/gotenberg.go internal/platform/render/gotenberg/client.go
git commit -m "feat(gotenberg): add Gotenberg HTTP client for DOCX-to-PDF conversion"
```

---

### Task 2: Add Gotenberg to Docker Compose and .env

**Files:**
- Modify: `deploy/compose/docker-compose.yml`
- Modify: `.env`

- [ ] **Step 1: Add Gotenberg service to docker-compose.yml**

Add after the `carbone` service block (before `api`):

```yaml
  gotenberg:
    image: gotenberg/gotenberg:8
    container_name: metaldocs-gotenberg
    restart: unless-stopped
    ports:
      - "3000:3000"
    healthcheck:
      test: ["CMD-SHELL", "curl -sf http://127.0.0.1:3000/health || exit 1"]
      interval: 5s
      timeout: 5s
      retries: 10
      start_period: 10s
```

- [ ] **Step 2: Add env var to .env**

Add to `.env`:

```
METALDOCS_GOTENBERG_URL=http://localhost:3000
```

- [ ] **Step 3: Add env var to API service environment in docker-compose.yml**

In the `api` service `environment` block, add:

```yaml
      METALDOCS_GOTENBERG_URL: ${METALDOCS_GOTENBERG_URL}
```

- [ ] **Step 4: Commit**

```bash
git add deploy/compose/docker-compose.yml .env
git commit -m "infra(gotenberg): add Gotenberg service to docker-compose"
```

---

### Task 3: Wire Gotenberg into Service and replace Carbone in convertDocxToPDF

**Files:**
- Modify: `internal/modules/documents/application/service.go`
- Modify: `internal/modules/documents/application/service_content_docx.go`
- Modify: `internal/platform/bootstrap/api.go`
- Modify: `apps/api/cmd/metaldocs-api/main.go`

- [ ] **Step 1: Add Gotenberg field and builder to Service struct**

In `internal/modules/documents/application/service.go`, add to the `Service` struct:

```go
gotenbergClient  *gotenberg.Client
```

Add import: `"metaldocs/internal/platform/render/gotenberg"`

Add builder method:

```go
func (s *Service) WithGotenberg(client *gotenberg.Client) *Service {
	s.gotenbergClient = client
	return s
}
```

- [ ] **Step 2: Replace `convertDocxToPDF` to use Gotenberg**

In `internal/modules/documents/application/service_content_docx.go`, replace the `convertDocxToPDF` method (lines 157-185):

```go
func (s *Service) convertDocxToPDF(ctx context.Context, content []byte, traceID string) ([]byte, error) {
	if s.gotenbergClient == nil {
		return nil, fmt.Errorf("gotenberg client not configured: PDF conversion unavailable")
	}
	return s.gotenbergClient.ConvertDocxToPDF(ctx, content)
}
```

Remove `"os"` from imports if no longer used.

- [ ] **Step 3: Wire Gotenberg in bootstrap and main.go**

In `internal/platform/bootstrap/api.go`:

Add to `APIDependencies`:

```go
GotenbergClient *gotenberg.Client
```

Add import: `"metaldocs/internal/platform/render/gotenberg"`

In `BuildAPIDependencies`, after `docgenClient := ...`:

```go
gotenbergCfg := config.LoadGotenbergConfig()
var gotenbergClient *gotenberg.Client
if gotenbergCfg.Enabled {
    gotenbergClient = gotenberg.NewClient(gotenbergCfg.URL)
}
```

Add to both return paths (postgres and memory): `GotenbergClient: gotenbergClient,`

In `apps/api/cmd/metaldocs-api/main.go`, add to the docService chain:

```go
WithGotenberg(deps.GotenbergClient).
```

- [ ] **Step 4: Verify compilation**

Run: `cd internal && go build ./... && cd ../apps/api && go build ./...`
Expected: No errors.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/application/service.go internal/modules/documents/application/service_content_docx.go internal/platform/bootstrap/api.go apps/api/cmd/metaldocs-api/main.go
git commit -m "feat(gotenberg): wire Gotenberg client and replace Carbone in convertDocxToPDF"
```

---

### Task 4: Extract shared `buildDocgenPayload` and unify save/render flows

This is the core change. Currently `SaveNativeContentAuthorized` calls `renderDocumentPDF` (Carbone). We rewrite it to use docgen + Gotenberg.

**Files:**
- Modify: `internal/modules/documents/application/service_document_runtime.go`
- Modify: `internal/modules/documents/application/service_content_native.go`

- [ ] **Step 1: Extract `buildDocgenPayload` in `service_document_runtime.go`**

Find the payload building code inside `ExportDocumentDocxAuthorized` and extract it into a shared method. Add this method before `ExportDocumentDocxAuthorized`:

```go
func (s *Service) buildDocgenPayload(ctx context.Context, doc domain.Document, schema domain.DocumentProfileSchemaVersion, version domain.Version) (docgen.RenderPayload, error) {
	schemaMap, err := toDocgenSchema(schema.ContentSchema)
	if err != nil {
		return docgen.RenderPayload{}, err
	}

	ownerName := s.resolveUserDisplayName(ctx, doc.OwnerID)
	approverName, approvedAt := s.resolveLatestApproval(ctx, doc.ID)

	payload := docgen.RenderPayload{
		DocumentType: firstNonEmpty(doc.DocumentType, doc.DocumentProfile),
		DocumentCode: doc.DocumentCode,
		Title:        doc.Title,
		Version:      fmt.Sprintf("%d", version.Number),
		Status:       doc.Status,
		Schema:       schemaMap,
		Values:       cloneRuntimeValues(version.Values),
		Metadata: &docgen.RenderMetadata{
			ElaboradoPor: ownerName,
			AprovadoPor:  approverName,
			CreatedAt:    doc.CreatedAt.Format("2006-01-02"),
			ApprovedAt:   approvedAt,
		},
	}

	versions, err := s.repo.ListVersions(ctx, doc.ID)
	if err == nil && len(versions) > 0 {
		revisions := make([]docgen.RenderRevision, 0, len(versions))
		for _, v := range versions {
			summary := v.ChangeSummary
			if summary == "" && v.Number == 1 {
				summary = "Criação do documento"
			}
			revisions = append(revisions, docgen.RenderRevision{
				Versao:    fmt.Sprintf("%d", v.Number),
				Data:      v.CreatedAt.Format("2006-01-02"),
				Descricao: summary,
				Por:       ownerName,
			})
		}
		payload.Revisions = revisions
	}

	return payload, nil
}
```

- [ ] **Step 2: Simplify `ExportDocumentDocxAuthorized` to use shared function**

Replace the body of `ExportDocumentDocxAuthorized`:

```go
func (s *Service) ExportDocumentDocxAuthorized(ctx context.Context, documentID, traceID string) ([]byte, error) {
	if s.docgenClient == nil {
		return nil, domain.ErrRenderUnavailable
	}

	bundle, err := s.GetDocumentRuntimeBundle(ctx, documentID)
	if err != nil {
		return nil, err
	}

	payload, err := s.buildDocgenPayload(ctx, bundle.Document, bundle.Schema, bundle.Version)
	if err != nil {
		return nil, err
	}

	return s.docgenClient.Generate(ctx, payload, traceID)
}
```

- [ ] **Step 3: Add `generateDocxBytes` helper for save/render flows**

Add this method to `service_document_runtime.go`:

```go
func (s *Service) generateDocxBytes(ctx context.Context, doc domain.Document, version domain.Version, content map[string]any, traceID string) ([]byte, error) {
	if s.docgenClient == nil {
		return nil, domain.ErrRenderUnavailable
	}

	schema, err := s.resolveActiveProfileSchema(ctx, doc.DocumentProfile)
	if err != nil {
		return nil, err
	}

	// Build a version with the content as Values for the payload builder
	versionWithValues := version
	if len(content) > 0 {
		versionWithValues.Values = content
	}

	payload, err := s.buildDocgenPayload(ctx, doc, schema, versionWithValues)
	if err != nil {
		return nil, err
	}

	return s.docgenClient.Generate(ctx, payload, traceID)
}
```

- [ ] **Step 4: Rewrite `SaveNativeContentAuthorized` to use docgen + Gotenberg**

In `service_content_native.go`, replace lines 90-98 (the `renderDocumentPDF` call and PDF storage) with:

```go
	// Generate DOCX via docgen
	docxBytes, err := s.generateDocxBytes(ctx, doc, version, cmd.Content, cmd.TraceID)
	if err != nil {
		return domain.Version{}, err
	}
	docxKey := documentContentStorageKey(doc.ID, next, "docx")
	if err := s.attachmentStore.Save(ctx, docxKey, docxBytes); err != nil {
		return domain.Version{}, err
	}
	version.DocxStorageKey = docxKey

	// Convert DOCX to PDF via Gotenberg
	pdfBytes, err := s.convertDocxToPDF(ctx, docxBytes, cmd.TraceID)
	if err != nil {
		// Cleanup orphaned DOCX on conversion failure
		_ = s.attachmentStore.Delete(ctx, docxKey)
		return domain.Version{}, err
	}
	pdfKey := documentContentStorageKey(doc.ID, next, "pdf")
	if err := s.attachmentStore.Save(ctx, pdfKey, pdfBytes); err != nil {
		_ = s.attachmentStore.Delete(ctx, docxKey)
		return domain.Version{}, err
	}
	version.PdfStorageKey = pdfKey
```

Note: The existing code after this block already has `s.repo.SaveVersion` with cleanup on failure. Update the cleanup to also delete the DOCX: change `_ = s.attachmentStore.Delete(ctx, pdfKey)` to also delete `docxKey`.

- [ ] **Step 5: Rewrite `RenderContentPDFAuthorized` default branch**

In `service_content_native.go`, replace the `default` case in `RenderContentPDFAuthorized` (lines 169-181). Instead of calling `renderDocumentPDF` (Carbone), generate DOCX first then convert:

```go
	default:
		content := version.NativeContent
		if len(content) == 0 && strings.TrimSpace(version.Content) != "" {
			var parsed map[string]any
			if err := json.Unmarshal([]byte(version.Content), &parsed); err == nil {
				content = parsed
			}
		}
		// If DOCX is cached, load it; otherwise generate via docgen
		var docxBytes []byte
		if strings.TrimSpace(version.DocxStorageKey) != "" {
			docxBytes, err = s.OpenContentStorage(ctx, version.DocxStorageKey)
			if err != nil {
				return domain.Version{}, err
			}
		} else {
			docxBytes, err = s.generateDocxBytes(ctx, doc, version, content, traceID)
			if err != nil {
				return domain.Version{}, err
			}
			// Cache the generated DOCX
			docxKey := documentContentStorageKey(doc.ID, version.Number, "docx")
			if saveErr := s.attachmentStore.Save(ctx, docxKey, docxBytes); saveErr == nil {
				_ = s.repo.UpdateVersionDocx(ctx, doc.ID, version.Number, docxKey)
			}
		}
		pdfBytes, err = s.convertDocxToPDF(ctx, docxBytes, traceID)
		if err != nil {
			return domain.Version{}, err
		}
```

- [ ] **Step 6: Verify compilation**

Run: `cd internal && go build ./...`
Expected: No errors.

- [ ] **Step 7: Commit**

```bash
git add internal/modules/documents/application/service_document_runtime.go internal/modules/documents/application/service_content_native.go
git commit -m "feat(docgen): unify save/render/export flows into single docgen pipeline"
```

---

### Task 5: Delete all Carbone code

**Files:**
- Delete: `internal/platform/render/carbone/client.go`
- Delete: `internal/platform/render/carbone/registry.go`
- Delete: `internal/platform/config/carbone.go`
- Modify: `internal/modules/documents/application/service.go`
- Modify: `internal/modules/documents/application/service_content_native.go`
- Modify: `internal/platform/bootstrap/api.go`
- Modify: `apps/api/cmd/metaldocs-api/main.go`
- Modify: `internal/modules/documents/delivery/http/handler_content.go`
- Modify: `internal/modules/documents/delivery/http/handler.go`

- [ ] **Step 1: Remove Carbone fields from Service struct**

In `service.go`, remove:
- `carboneClient *carbone.Client` field
- `carboneTemplates *carbone.TemplateRegistry` field
- `WithCarbone()` method
- `"metaldocs/internal/platform/render/carbone"` import

- [ ] **Step 2: Delete Carbone rendering functions from `service_content_native.go`**

Remove these functions entirely:
- `renderProfileTemplate()` (lines 243-257)
- `renderDocumentPDF()` (lines 259-264)
- `buildDocumentTemplateData()` (lines 266-288)
- `RenderProfileTemplateDocx()` (line 199-201)
- `RenderDocumentTemplateDocxAuthorized()` (lines 203-220)

Also remove helper functions that were only used by Carbone: `cloneContentMap()`, `cloneEtapaBodies()`, `mergeEtapaBodyBlocks()` — if they exist and are unused after the removal.

- [ ] **Step 3: Deprecate template download handlers and update OpenAPI**

In `handler_content.go`, replace the `handleDocumentProfileTemplateDocx` and `handleDocumentTemplateDocx` function bodies with a 501 + JSON error response:

```go
func (h *Handler) handleDocumentProfileTemplateDocx(w http.ResponseWriter, r *http.Request, profileCode string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"error":{"code":"TEMPLATE_DEPRECATED","message":"Carbone template rendering has been removed. Use the content builder instead."}}`))
}
```

In `api/openapi/v1/openapi.yaml`, update both template routes (`/document-profiles/{profileCode}/template/docx` and `/documents/{documentId}/template/docx`) to mark them as deprecated and document the 501 response:

```yaml
    deprecated: true
    responses:
      '501':
        description: Template rendering removed — use content builder
```

- [ ] **Step 3b: Replace Carbone health check with Gotenberg**

In `internal/platform/bootstrap/api.go`, find the Carbone dependency check registered with the `RuntimeStatusProvider`. Replace it with a Gotenberg health check:

```go
gotenbergCheck := observability.DependencyCheck{
    Name: "gotenberg",
    Check: func(ctx context.Context) error {
        if gotenbergClient == nil {
            return fmt.Errorf("gotenberg not configured")
        }
        resp, err := http.Get(gotenbergCfg.URL + "/health")
        if err != nil {
            return err
        }
        defer resp.Body.Close()
        if resp.StatusCode != http.StatusOK {
            return fmt.Errorf("gotenberg unhealthy: %d", resp.StatusCode)
        }
        return nil
    },
}
```

Register it where the Carbone check was previously registered.

- [ ] **Step 4: Remove Carbone from bootstrap and main.go**

In `bootstrap/api.go`:
- Remove `CarboneClient` and `CarboneTemplates` from `APIDependencies`
- Remove `carboneClient := carbone.NewClient(carboneCfg)` and `carboneRegistry` creation
- Remove Carbone imports
- Remove Carbone from both return paths (postgres and memory)
- Remove `carboneCfg` parameter from `BuildAPIDependencies` signature

In `main.go`:
- Remove `carboneCfg := config.LoadCarboneConfig()`
- Remove `carboneCfg` from `BuildAPIDependencies` call
- Remove `.WithCarbone(deps.CarboneClient, deps.CarboneTemplates)` from service chain
- Remove Carbone config import

- [ ] **Step 5: Delete Carbone package and config**

```bash
rm -rf internal/platform/render/carbone/
rm internal/platform/config/carbone.go
```

- [ ] **Step 6: Verify compilation**

Run: `cd internal && go build ./... && cd ../apps/api && go build ./...`
Expected: No errors.

- [ ] **Step 7: Remove Carbone from Docker Compose**

In `deploy/compose/docker-compose.yml`:
- Delete the entire `carbone` service block (image, container_name, ports, volumes, environment)
- Remove `METALDOCS_CARBONE_API_URL` from the api service environment
- Remove `carbone: condition: service_started` from api depends_on

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "refactor: remove all Carbone code, config, and infrastructure"
```

---

### Task 6: Verify the `UpdateVersionDocx` repo method exists

The rewritten `RenderContentPDFAuthorized` calls `s.repo.UpdateVersionDocx(ctx, docID, versionNumber, docxKey)` to cache the DOCX key. This method may not exist.

**Files:**
- Possibly modify: `internal/modules/documents/domain/port.go`
- Possibly modify: `internal/modules/documents/infrastructure/postgres/repository.go`

- [ ] **Step 1: Check if `UpdateVersionDocx` exists**

Run: `grep -rn "UpdateVersionDocx" internal/modules/documents/`

If it exists, skip to Step 4. If not, continue.

- [ ] **Step 2: Add to Repository interface**

In `domain/port.go`, add to the `Repository` interface:

```go
UpdateVersionDocx(ctx context.Context, documentID string, versionNumber int, docxStorageKey string) error
```

- [ ] **Step 3: Implement in postgres repository**

In `infrastructure/postgres/repository.go`, add:

```go
func (r *Repository) UpdateVersionDocx(ctx context.Context, documentID string, versionNumber int, docxStorageKey string) error {
	const q = `UPDATE metaldocs.document_versions SET docx_storage_key = $3 WHERE document_id = $1 AND version_number = $2`
	_, err := r.db.ExecContext(ctx, q, documentID, versionNumber, docxStorageKey)
	return err
}
```

Also add to the memory repository if one exists.

- [ ] **Step 4: Verify compilation**

Run: `cd internal && go build ./...`

- [ ] **Step 5: Commit (only if changes were needed)**

```bash
git add internal/modules/documents/domain/port.go internal/modules/documents/infrastructure/postgres/repository.go
git commit -m "feat(repo): add UpdateVersionDocx method for DOCX storage key caching"
```

---

### Task 7: Focused regression tests

**Files:**
- Create: `internal/modules/documents/application/service_content_native_test.go`
- Create: `internal/modules/documents/delivery/http/handler_content_test.go`

- [ ] **Step 1: Test SaveNativeContentAuthorized cleanup on PDF conversion failure**

Write a test that mocks `gotenbergClient.ConvertDocxToPDF` to return an error. Assert that the orphaned DOCX is deleted from the attachment store (verify `attachmentStore.Delete` was called with the docx key).

- [ ] **Step 2: Test SaveNativeContentAuthorized cleanup on version persistence failure**

Write a test that mocks `repo.SaveVersion` to return an error. Assert that both the DOCX and PDF are deleted from the attachment store.

- [ ] **Step 3: Test template download routes return 501 JSON**

Write an HTTP handler test that calls `handleDocumentProfileTemplateDocx` and asserts:
- Status code: 501
- Content-Type: application/json
- Body contains `TEMPLATE_DEPRECATED` error code

- [ ] **Step 4: Verify tests pass**

Run: `cd internal && go test ./modules/documents/... -v`
Expected: All tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/application/service_content_native_test.go internal/modules/documents/delivery/http/handler_content_test.go
git commit -m "test: add regression tests for cleanup semantics and deprecated template routes"
```

---

### Task 8: End-to-end smoke test

- [ ] **Step 1: Start Gotenberg**

Run: `docker run -d --name metaldocs-gotenberg -p 3000:3000 gotenberg/gotenberg:8`

Verify: `curl -s http://localhost:3000/health` — should return `{"status":"up"}`

- [ ] **Step 2: Restart local API**

Run your `run_metaldocs` script to restart the API with the new code.

- [ ] **Step 3: Verify schema is served**

Create or open a PO document. In the content builder, verify 8 sections appear.

- [ ] **Step 4: Test save + PDF preview**

1. Fill in `objetivo` in Section 2
2. Wait for auto-save (3s debounce)
3. Check: Does the preview panel show a PDF? Check Network tab for `POST /content/native` response — should have `pdfUrl`
4. Check: API logs should NOT mention Carbone

- [ ] **Step 5: Test "Gerar PDF" button**

1. Click "Gerar PDF"
2. Check: `POST /content/render-pdf` should succeed
3. Preview panel updates with new PDF

- [ ] **Step 6: Test "Exportar .docx" button**

1. Click "Exportar .docx"
2. DOCX downloads
3. Open it — should have Section 1 (metadata), Sections 2-9 (content), Section 10 (revisions)

- [ ] **Step 7: Verify no Carbone references in logs**

Run: `grep -i carbone` in the API output — should find nothing (or only the "carbone bootstrap degraded" from before restart, not from new requests).

- [ ] **Step 8: Commit any fixes**

```bash
git add -A
git commit -m "fix: address issues from carbone removal smoke test"
```
