# Audit Remediation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the 4 MAJOR issues and 1 MINOR issue identified by the implementation audit.

**Architecture:** Targeted fixes to existing code — no new abstractions. Fix save pipeline ordering, add Gotenberg health check, update OpenAPI, delete dead code.

**Tech Stack:** Go 1.24, YAML (OpenAPI)

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/modules/documents/application/service_content_native.go` | Modify | Fix save pipeline: append pending revision to payload |
| `internal/modules/documents/application/service_document_runtime.go` | Modify | Accept optional pending revision in buildDocgenPayload; delete legacy export |
| `internal/platform/bootstrap/api.go` | Modify | Add Gotenberg health check to RuntimeStatusProvider |
| `api/openapi/v1/openapi.yaml` | Modify | Deprecate template routes, add 501 response |

---

### Task 1: Fix save pipeline — append pending revision to docgen payload

The problem: `SaveNativeContentAuthorized` calls `generateDocxBytes` (which calls `buildDocgenPayload`) BEFORE `SaveVersion`. So `buildDocgenPayload` reads `ListVersions` and the current save isn't in the list yet — Section 10 is missing the current revision.

The fix: Pass the pending version info into `buildDocgenPayload` so it appends it to the revisions list.

**Files:**
- Modify: `internal/modules/documents/application/service_document_runtime.go`
- Modify: `internal/modules/documents/application/service_content_native.go`

- [ ] **Step 1: Add `pendingRevision` parameter to `buildDocgenPayload`**

In `service_document_runtime.go`, change the signature of `buildDocgenPayload` from:

```go
func (s *Service) buildDocgenPayload(ctx context.Context, doc domain.Document, schema domain.DocumentProfileSchemaVersion, version domain.Version) (docgen.RenderPayload, error) {
```

To:

```go
func (s *Service) buildDocgenPayload(ctx context.Context, doc domain.Document, schema domain.DocumentProfileSchemaVersion, version domain.Version, pendingRevision *docgen.RenderRevision) (docgen.RenderPayload, error) {
```

After building the `revisions` slice from `ListVersions`, append the pending revision if non-nil:

```go
	if pendingRevision != nil {
		payload.Revisions = append(payload.Revisions, *pendingRevision)
	}
```

Add this right before `return payload, nil`.

- [ ] **Step 2: Update all callers of `buildDocgenPayload`**

In `ExportDocumentDocxAuthorized` (line ~232), pass `nil` for pendingRevision:

```go
payload, err := s.buildDocgenPayload(ctx, bundle.Document, bundle.Schema, bundle.Version, nil)
```

In `generateDocxBytes` (line ~255), pass `nil` for pendingRevision:

```go
payload, err := s.buildDocgenPayload(ctx, doc, schema, versionWithValues, nil)
```

- [ ] **Step 3: Pass pending revision from `SaveNativeContentAuthorized`**

In `service_content_native.go`, the call to `generateDocxBytes` is at line 90. We need to modify `generateDocxBytes` to accept an optional pending revision, OR build the pending revision inline and pass it through.

The simplest approach: add `pendingRevision *docgen.RenderRevision` parameter to `generateDocxBytes`:

```go
func (s *Service) generateDocxBytes(ctx context.Context, doc domain.Document, version domain.Version, content map[string]any, traceID string, pendingRevision *docgen.RenderRevision) ([]byte, error) {
```

Update the call in `generateDocxBytes` to pass it through:

```go
payload, err := s.buildDocgenPayload(ctx, doc, schema, versionWithValues, pendingRevision)
```

In `SaveNativeContentAuthorized`, build the pending revision and pass it:

```go
	pending := &docgen.RenderRevision{
		Versao:    fmt.Sprintf("%d", next),
		Data:      now.Format("2006-01-02"),
		Descricao: fmt.Sprintf("Content version %d", next),
		Por:       doc.OwnerID,
	}
	docxBytes, err := s.generateDocxBytes(ctx, doc, version, contentPayload, cmd.TraceID, pending)
```

Update the other caller of `generateDocxBytes` in `RenderContentPDFAuthorized` (default branch) to pass `nil`:

```go
docxBytes, err = s.generateDocxBytes(ctx, doc, version, content, traceID, nil)
```

- [ ] **Step 4: Verify compilation**

Run: `cd internal && go build ./...`

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/application/service_document_runtime.go internal/modules/documents/application/service_content_native.go
git commit -m "fix(docgen): include pending revision in save-generated DOCX payload"
```

---

### Task 2: Add Gotenberg health check to RuntimeStatusProvider

**Files:**
- Modify: `internal/platform/bootstrap/api.go`

- [ ] **Step 1: Create Gotenberg dependency check and pass to StatusProvider**

In `BuildAPIDependencies`, after the Gotenberg client creation, build a dependency check. Then pass it to the status providers.

For the postgres path (around line 98), change:

```go
StatusProvider: observability.NewPostgresRuntimeStatusProvider(db, repoMode, attachmentsCfg.Provider, authn.Enabled()),
```

To:

```go
StatusProvider: observability.NewPostgresRuntimeStatusProvider(db, repoMode, attachmentsCfg.Provider, authn.Enabled(), gotenbergHealthCheck(gotenbergCfg)),
```

For the memory path (around line 132), change similarly:

```go
StatusProvider: observability.NewStaticRuntimeStatusProvider(repoMode, attachmentsCfg.Provider, authn.Enabled(), gotenbergHealthCheck(gotenbergCfg)),
```

Add the helper function at the bottom of the file:

```go
func gotenbergHealthCheck(cfg config.GotenbergConfig) observability.DependencyCheck {
	return observability.DependencyCheck{
		Name: "gotenberg",
		Check: func(ctx context.Context) (observability.DependencyCheckResult, error) {
			if !cfg.Enabled || cfg.URL == "" {
				return observability.DependencyCheckResult{
					Status: "skipped",
					Detail: "gotenberg not configured",
				}, nil
			}
			client := &http.Client{Timeout: 2 * time.Second}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL+"/health", nil)
			if err != nil {
				return observability.DependencyCheckResult{}, err
			}
			resp, err := client.Do(req)
			if err != nil {
				return observability.DependencyCheckResult{}, err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return observability.DependencyCheckResult{}, fmt.Errorf("gotenberg unhealthy: status %d", resp.StatusCode)
			}
			return observability.DependencyCheckResult{
				Status: "up",
				Detail: cfg.URL,
			}, nil
		},
	}
}
```

Add imports if needed: `"net/http"`, `"time"`.

- [ ] **Step 2: Verify compilation**

Run: `cd internal && go build ./... && cd ../apps/api && go build ./...`

- [ ] **Step 3: Commit**

```bash
git add internal/platform/bootstrap/api.go
git commit -m "fix(health): add Gotenberg dependency check to runtime status provider"
```

---

### Task 3: Update OpenAPI for deprecated template routes

**Files:**
- Modify: `api/openapi/v1/openapi.yaml`

- [ ] **Step 1: Deprecate profile template route**

At `api/openapi/v1/openapi.yaml` line 595, add `deprecated: true` and a 501 response:

```yaml
  /document-profiles/{profileCode}/template/docx:
    get:
      deprecated: true
      summary: "[DEPRECATED] Template DOCX rendering removed — use content builder"
      parameters:
        - name: profileCode
          in: path
          required: true
          schema:
            type: string
      responses:
        '501':
          description: Template rendering removed
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ApiErrorEnvelope'
```

Remove the old 200/400/401 responses for this route.

- [ ] **Step 2: Deprecate document template route**

At line 1727, do the same for `/documents/{documentId}/template/docx`:

```yaml
  /documents/{documentId}/template/docx:
    get:
      deprecated: true
      summary: "[DEPRECATED] Template DOCX rendering removed — use content builder"
      parameters:
        - $ref: '#/components/parameters/DocumentId'
      responses:
        '501':
          description: Template rendering removed
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ApiErrorEnvelope'
```

Remove the old 200/401/403/404 responses for this route.

- [ ] **Step 3: Commit**

```bash
git add api/openapi/v1/openapi.yaml
git commit -m "docs(openapi): deprecate template DOCX routes with 501 response"
```

---

### Task 4: Delete dead legacy export code

**Files:**
- Modify: `internal/modules/documents/application/service_document_runtime.go`

- [ ] **Step 1: Delete `exportDocumentDocxAuthorizedLegacy`**

Find the function `exportDocumentDocxAuthorizedLegacy` (line ~263) and delete it entirely — it's unreachable dead code.

Also delete any helper functions that were only used by this legacy function (check for `toRuntimeMapSlice`, `toRuntimeString`, `toRuntimeStringSlice`, `normalizeDocgenScalarType`, `toDocgenField`, `toDocgenSection` — if they're still used by `toDocgenSchema` or `buildDocgenPayload`, keep them).

- [ ] **Step 2: Verify compilation**

Run: `cd internal && go build ./...`

- [ ] **Step 3: Commit**

```bash
git add internal/modules/documents/application/service_document_runtime.go
git commit -m "chore: delete dead exportDocumentDocxAuthorizedLegacy code"
```
