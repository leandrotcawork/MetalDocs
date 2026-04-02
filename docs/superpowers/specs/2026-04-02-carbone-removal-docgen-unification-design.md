# Carbone Removal & Docgen Pipeline Unification

**Date:** 2026-04-02
**Status:** Approved
**Scope:** Remove all Carbone dependencies, unify docgen pipeline, add Gotenberg for PDF conversion

---

## Problem

The MetalDocs backend has two parallel document rendering systems that diverged during the migration from Carbone to docgen/docx-js:

1. **Save/Preview flow** (`service_content_native.go`) uses docgen for DOCX but falls back to Carbone for PDF conversion. The payload builder (`ProjectDocgenPayload` in `docgen_projection.go`) uses old Carbone-era field paths (`content["process"]["etapas"]`) that don't match the new PO v3 schema (`content["etapas"]["etapas"]`).

2. **Export flow** (`service_document_runtime.go`) uses a completely separate docgen pipeline with the correct schema resolution, metadata enrichment, and revision history — but this code is only reached via the DOCX export endpoint, not the save/preview path.

3. **Carbone** is used for all DOCX-to-PDF conversion and has a broken direct-rendering fallback (templates no longer exist). It's unreliable infrastructure with no value in the new architecture.

### The 5 Specific Bugs

| # | Bug | Location | Root Cause |
|---|-----|----------|------------|
| 1 | `ProjectDocgenPayload()` uses old schema field paths | `docgen_projection.go` | Extracts `content["process"]["etapas"]` — wrong keys for PO v3 |
| 2 | Two different docgen client interfaces | `service.go` has `DocgenClient` interface; `service_document_runtime.go` uses `*docgen.Client` | Duplicate abstractions from parallel development |
| 3 | `convertDocxToPDF()` always falls back to Carbone | `service_content_docx.go:218` | No `DocxConverter` wired in `main.go` |
| 4 | `RenderContentPDFAuthorized` has broken Carbone fallback | `service_content_native.go:200` | Carbone templates deleted, `renderDocumentPDF()` always fails |
| 5 | Export enrichment (metadata/revisions) not in save/preview path | `service_document_runtime.go` vs `service_content_native.go` | Two separate pipeline implementations |

---

## Solution Overview

Three changes that fix all 5 bugs:

1. **Unify docgen pipeline** — Extract shared `buildDocgenPayload()` function used by both save/preview and export flows. Delete the old `ProjectDocgenPayload` and consolidate to one docgen client.
2. **Add Gotenberg for PDF conversion** — New HTTP client implementing the existing `DocxConverter` interface. Wire it in `main.go`. All DOCX-to-PDF goes through Gotenberg.
3. **Delete all Carbone code** — Remove the package, config, Docker container, and all references. Clean break.

---

## 1. Unified Docgen Pipeline

### Shared payload builder

Extract from `ExportDocumentDocxAuthorized` into a reusable function:

```
buildDocgenPayload(doc Document, schema DocumentProfileSchemaVersion, version Version, metadata, revisions)
    → docgen.RenderPayload
```

This function:
- Resolves the type schema via `toDocgenSchema(schema.ContentSchema)`
- Populates `DocumentType`, `DocumentCode`, `Title`, `Version`, `Status`
- Populates `Metadata` (elaboradoPor, aprovadoPor, createdAt, approvedAt)
- Populates `Revisions` from version history
- Populates `Values` from version values

### Callers

**`SaveNativeContentAuthorized`:**
1. Save native content to DB
2. Resolve schema via `resolveActiveProfileSchema`
3. Call `buildDocgenPayload` → `docgenClient.Generate()` → DOCX bytes
4. Store DOCX in MinIO
5. Call `gotenbergClient.ConvertDocxToPDF()` → PDF bytes
6. Store PDF in MinIO
7. Return `{ pdfUrl, version }`

**`RenderContentPDFAuthorized`:**
1. Load cached DOCX from storage (if exists)
2. If no DOCX: resolve schema, call `buildDocgenPayload` → `docgenClient.Generate()` → store DOCX
3. Call `gotenbergClient.ConvertDocxToPDF()` → PDF bytes
4. Store PDF in MinIO
5. Return `{ pdfUrl, version }`

**`ExportDocumentDocxAuthorized`:** (stays as-is, already uses correct pipeline)
1. Call `buildDocgenPayload` → `docgenClient.Generate()` → DOCX bytes
2. Return DOCX bytes

### Code to delete

- `internal/modules/documents/application/docgen_projection.go` — entire file (`ProjectDocgenPayload`, `DocgenPayload` type)
- `DocgenClient` interface in `service.go` (the old one with `Generate(ctx, traceID, payload)` signature)
- `s.docgen` field on Service — consolidate to `s.docgenClient` only
- `WithDocgen()` builder — consolidate to `WithDocgenClient()` only
- `renderDocumentDocx()` in `service_content_docx.go` — replaced by shared `buildDocgenPayload` + `docgenClient.Generate`
- `renderDocumentPDF()` in `service_content_native.go` — broken Carbone path
- `renderProfileTemplate()` in `service_content_native.go` — Carbone template rendering
- `buildDocumentTemplateData()` in `service_content_native.go` — old template data builder
- `RenderProfileTemplateDocx()` — Carbone-based blank template download

### Single docgen client

After cleanup, only `*docgen.Client` (from `internal/platform/render/docgen/`) remains. The `Service` struct has one field: `docgenClient *docgen.Client`.

---

## 2. Gotenberg Integration

### Infrastructure

Add to `docker-compose.yml`:

```yaml
gotenberg:
  image: gotenberg/gotenberg:8
  container_name: metaldocs-gotenberg
  restart: unless-stopped
  ports:
    - "3000:3000"
```

Add to `.env`:

```
METALDOCS_GOTENBERG_URL=http://localhost:3000
```

### Go Client

Create `internal/platform/render/gotenberg/client.go`:

```go
type Client struct {
    baseURL    string
    httpClient *http.Client
}

func NewClient(baseURL string) *Client

func (c *Client) ConvertDocxToPDF(ctx context.Context, docxContent []byte) ([]byte, error)
```

The `ConvertDocxToPDF` method:
- POST multipart form to `{baseURL}/forms/libreoffice/convert`
- Attach DOCX bytes as `files` field with filename `document.docx`
- Return response body as PDF bytes

### Wiring

The `Service` struct already has a `DocxConverter` interface:

```go
type DocxConverter interface {
    Convert(ctx context.Context, docxContent []byte, traceID string) ([]byte, error)
}
```

Create a thin adapter (`GotenbergAdapter`) that wraps `gotenberg.Client` and implements `DocxConverter`. Or adjust the interface to match — either way, the `convertDocxToPDF` method in `service_content_docx.go` already checks `s.docxConverter != nil` as its primary path.

In `main.go`:

```go
gotenbergClient := gotenberg.NewClient(gotenbergCfg.URL)
docService.WithDocxConverter(gotenbergAdapter)
```

### Config

Create `internal/platform/config/gotenberg.go`:

```go
type GotenbergConfig struct {
    URL string
}

func LoadGotenbergConfig() GotenbergConfig {
    return GotenbergConfig{
        URL: envOrDefault("METALDOCS_GOTENBERG_URL", "http://localhost:3000"),
    }
}
```

---

## 3. Carbone Removal

### Go code to delete

| File/Package | What | Why |
|-------------|------|-----|
| `internal/platform/render/carbone/` | Entire package | No longer used |
| `internal/platform/config/carbone.go` | Carbone config | No longer needed |
| `docgen_projection.go` | `ProjectDocgenPayload` | Old schema field paths |
| `service.go` | `carboneClient`, `carboneTemplates` fields, `WithCarbone()`, old `DocgenClient` interface, `WithDocgen()` | Replaced by unified pipeline |
| `service_content_native.go` | `renderProfileTemplate()`, `renderDocumentPDF()`, `buildDocumentTemplateData()`, `RenderProfileTemplateDocx()` | Carbone rendering functions |
| `service_content_docx.go` | Carbone fallback in `convertDocxToPDF()` | Replaced by Gotenberg |

### Bootstrap/main.go

Remove:
- `carboneClient := carbone.NewClient(carboneCfg)`
- `carboneRegistry, err := carbone.BootstrapTemplates(...)`
- `deps.CarboneClient`, `deps.CarboneTemplates` from `APIDependencies`
- `.WithCarbone(deps.CarboneClient, deps.CarboneTemplates)` from service construction
- Carbone config loading
- Carbone health check from `RuntimeStatusProvider`

### Infrastructure

- Remove `carbone` service from `docker-compose.yml`
- Remove `carbone/templates` and `carbone/renders` volume mounts
- Remove `METALDOCS_CARBONE_API_URL` from `.env`
- The `carbone/` directory at repo root can be deleted (templates, renders)

### Handler cleanup

- Remove or stub the profile template DOCX download handler (`handleDocumentProfileTemplateDocx`) — this used Carbone. Either delete the route or return 501 Not Implemented.

---

## 4. Updated Data Flow (After Changes)

### Save + Auto-save

```
POST /content/native
    → SaveNativeContentAuthorized()
        → resolveActiveProfileSchema() → schema with type schema sections
        → buildDocgenPayload(doc, schema, version, metadata, revisions)
        → docgenClient.Generate(payload) → DOCX bytes
        → Store DOCX in MinIO
        → gotenbergClient.ConvertDocxToPDF(docxBytes) → PDF bytes
        → Store PDF in MinIO
        → Return { pdfUrl, version }
```

### Render PDF

```
POST /content/render-pdf
    → RenderContentPDFAuthorized()
        → Load DOCX from storage OR generate via buildDocgenPayload + docgenClient
        → gotenbergClient.ConvertDocxToPDF(docxBytes) → PDF bytes
        → Store PDF in MinIO
        → Return { pdfUrl }
```

### Export DOCX

```
POST /export/docx
    → ExportDocumentDocxAuthorized()
        → buildDocgenPayload()
        → docgenClient.Generate(payload) → DOCX bytes
        → Return DOCX download
```

### No Carbone in any path.

---

## 5. What Changes vs What Stays

### Changes

| Component | Change |
|-----------|--------|
| `service.go` | Remove Carbone fields/methods, remove old DocgenClient interface, keep only `docgenClient` and `docxConverter` |
| `service_content_native.go` | Rewrite `SaveNativeContentAuthorized` and `RenderContentPDFAuthorized` to use shared pipeline. Delete Carbone rendering functions. |
| `service_content_docx.go` | Remove Carbone fallback from `convertDocxToPDF`. Delete `renderDocumentDocx` (replaced by shared builder). |
| `service_document_runtime.go` | Extract `buildDocgenPayload` into shared function. `ExportDocumentDocxAuthorized` calls it. |
| `docgen_projection.go` | Delete entire file |
| `main.go` | Remove Carbone, add Gotenberg, wire `DocxConverter` |
| `bootstrap/api.go` | Remove Carbone deps, add Gotenberg config |
| `docker-compose.yml` | Remove Carbone, add Gotenberg |
| `internal/platform/render/carbone/` | Delete entire package |
| `internal/platform/render/gotenberg/` | Create new package |

### No changes

| Component | Why |
|-----------|-----|
| Frontend (`ContentBuilderView`, `DynamicEditor`, API client) | Same API shape, same response format |
| `apps/docgen/` (Express + docx-js) | Already correct — schema-driven generation |
| `internal/platform/render/docgen/` | Client and types stay as-is |
| Database schema | No changes |
| PreviewPanel | Same `pdfUrl` prop |

---

## 6. Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| Gotenberg container not running | `convertDocxToPDF` returns clear error; health check in compose |
| Old documents with Carbone-era native_content | New schema serves empty sections for old keys; no data loss |
| Profile template download feature breaks | Route returns 501 or redirect to alternative; feature was rarely used |
| Docgen Express service not running | Already handled — `docgenClient.Generate` returns `ErrRenderUnavailable` |
