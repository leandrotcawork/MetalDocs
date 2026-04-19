# Schema Runtime Platform Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current profile and template-centered document flow with a schema-driven runtime that powers document types, editor, preview, persistence, and `.docx` export from one canonical contract.

**Architecture:** Keep the existing `documents` module, workflow, auth, and approval flow, but introduce a new schema-runtime core inside that module. Migrate additively at the database layer, cut over the HTTP/API and frontend runtime, and make `apps/docgen` render `schema + values` instead of profile-specific payloads. Current data is dev-only, so the code may cut over cleanly without compatibility adapters for saved content.

**Tech Stack:** Go modular monolith, Postgres JSONB, OpenAPI v1, React + TypeScript + Zustand, Tiptap, Node.js + Express + `docx`

---

## Scope Check

This migration spans three subsystems, but they are tightly coupled around one canonical runtime contract:

- Go backend: schema persistence, validation, document values, API, export assembly
- Frontend: dynamic editor and preview from schema
- `apps/docgen`: schema-driven `.docx` renderer

This plan keeps them in one coordinated migration because they all depend on the same canonical schema contract and cannot produce a correct working slice independently. Commits still stay small and task-scoped.

## File Structure

### Backend: new runtime core inside `documents`

- Create: `docs/adr/0020-schema-runtime-document-platform.md`
  - Architecture decision for schema runtime, additive migration strategy, and v1 scope.
- Create: `internal/modules/documents/domain/schema_runtime.go`
  - Canonical Go structs for `DocumentTypeSchema`, `SectionDef`, `FieldDef`, rich blocks, and validation entrypoints.
- Create: `internal/modules/documents/domain/schema_runtime_errors.go`
  - Structured runtime validation errors such as `DOCUMENT_SCHEMA_INVALID_FIELD`.
- Create: `internal/modules/documents/application/service_schema_runtime.go`
  - `DocumentType` and `schema` read and write use cases.
- Create: `internal/modules/documents/application/service_document_runtime.go`
  - document-value read, save, editor bundle, preview payload, export payload assembly.
- Create: `internal/modules/documents/application/service_runtime_validation.go`
  - schema and document-value validation orchestration.
- Create: `internal/modules/documents/delivery/http/handler_runtime.go`
  - runtime endpoints for document types, editor bundle, save values, preview, export.
- Create: `internal/platform/config/docgen.go`
  - docgen base URL and timeout configuration.
- Create: `internal/platform/render/docgen/client.go`
  - Go HTTP client for `apps/docgen`.
- Create: `internal/platform/render/docgen/types.go`
  - schema + values payload contract for the renderer.
- Modify: `internal/modules/documents/domain/model.go`
  - remove profile-centric assumptions from new runtime paths and add `DocumentTypeKey`, `DocumentTypeVersion`, `ValuesJSON`.
- Modify: `internal/modules/documents/domain/port.go`
  - add document type, schema version, and document values repository methods.
- Modify: `internal/modules/documents/infrastructure/postgres/repository.go`
  - persist schema runtime tables and document values JSONB.
- Modify: `internal/modules/documents/infrastructure/memory/repository.go`
  - mirror runtime behavior for tests and local mode.
- Modify: `internal/modules/documents/delivery/http/handler.go`
  - register new routes and retire old profile-bundle/editor-bundle use in the cutover paths.
- Modify: `internal/platform/bootstrap/api.go`
  - wire docgen client config and runtime service dependencies.

### Database and seed data

- Create: `migrations/0048_init_document_type_runtime.sql`
  - new additive tables for `document_types`, `document_type_schema_versions`, and JSONB value storage.
- Create: `migrations/0049_grant_document_type_runtime_privileges.sql`
  - runtime grants.
- Create: `migrations/0050_seed_document_type_runtime.sql`
  - seed PO, IT, and RG in the new canonical schema format.

### API contract

- Modify: `api/openapi/v1/openapi.yaml`
  - add schema runtime endpoints and de-emphasize the old profile-specific bundle endpoints.

### Frontend runtime and editor

- Create: `frontend/apps/web/src/features/documents/runtime/schemaRuntimeTypes.ts`
  - canonical TS types for schema runtime and rich blocks.
- Create: `frontend/apps/web/src/features/documents/runtime/schemaRuntimeAdapters.ts`
  - adapters between API payloads and editor state.
- Create: `frontend/apps/web/src/features/documents/runtime/useSchemaDocumentEditor.ts`
  - editor data loading, save, preview, and export orchestration.
- Create: `frontend/apps/web/src/features/documents/runtime/DynamicEditor.tsx`
  - schema-driven editor shell.
- Create: `frontend/apps/web/src/features/documents/runtime/DynamicPreview.tsx`
  - schema-driven preview shell.
- Create: `frontend/apps/web/src/features/documents/runtime/fields/RichField.tsx`
  - Tiptap editor with image, table, lists, and formatting.
- Create: `frontend/apps/web/src/features/documents/runtime/fields/RepeatField.tsx`
  - repeat field renderer with nested runtime fields.
- Create: `frontend/apps/web/src/features/documents/runtime/fields/TableField.tsx`
  - structured table field renderer.
- Create: `frontend/apps/web/src/features/documents/runtime/fields/ScalarField.tsx`
  - text, textarea, date, number, select, checkbox fields.
- Create: `frontend/apps/web/src/features/documents/runtime/DynamicEditor.module.css`
- Create: `frontend/apps/web/src/features/documents/runtime/RichField.module.css`
- Modify: `frontend/apps/web/src/api/documents.ts`
  - runtime document type, bundle, save, and export endpoints.
- Modify: `frontend/apps/web/src/store/documents.store.ts`
  - schema runtime editor state and selected type state.
- Modify: `frontend/apps/web/src/components/content-builder/contentSchemaTypes.ts`
  - replace the current shallow schema types with imports from runtime types.
- Modify: `frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx`
  - switch from profile-specific form rendering to `DynamicEditor`.
- Modify: `frontend/apps/web/src/components/content-builder/preview/DocumentPreviewRenderer.tsx`
  - switch from template registry to `DynamicPreview`.
- Modify: `frontend/apps/web/src/components/content-builder/preview/templates/templateRegistry.ts`
  - remove usage from active flow and retire template-specific registry.
- Modify: `frontend/apps/web/package.json`
  - add Tiptap packages.

### Docgen

- Create: `apps/docgen/src/runtime/types.ts`
  - renderer payload types.
- Create: `apps/docgen/src/runtime/blocks.ts`
  - `rich` block to `docx` conversion.
- Create: `apps/docgen/src/runtime/renderSection.ts`
  - generic section renderer.
- Create: `apps/docgen/src/runtime/renderField.ts`
  - field dispatch by type.
- Modify: `apps/docgen/src/generate.ts`
  - accept schema runtime payload and orchestrate rendering.
- Modify: `apps/docgen/src/index.ts`
  - validate renderer request and return structured errors.
- Modify: `apps/docgen/scripts/sample-payload.json`
  - replace profile payload sample with schema runtime sample.
- Modify: `apps/docgen/scripts/harness.ps1`
  - keep the same minimal harness, but post the new sample payload.
- Modify: `apps/docgen/package.json`
  - add Tiptap-independent docgen-only dependencies if needed, keep harness scripts.

### Tests

- Modify: `tests/unit/documents_service_test.go`
  - schema validation, save values, export payload, document-level versioning.
- Modify: `tests/unit/documents_postgres_repository_test.go`
  - runtime schema and values persistence.
- Modify: `tests/unit/documents_http_handler_test.go`
  - runtime endpoints and authorization.
- Modify: `tests/contract/api_contract_smoke_test.go`
  - OpenAPI surface for runtime endpoints.

## Canonical Runtime Contract

The migration should implement this contract first and let all other tasks consume it:

```ts
interface DocumentTypeSchema {
  sections: SectionDef[]
}

interface SectionDef {
  key: string
  num: string
  title: string
  color?: string
  fields: FieldDef[]
}

type FieldDef =
  | { key: string; label: string; type: "text" | "textarea" | "number" | "date" | "select" | "checkbox" }
  | { key: string; label: string; type: "table"; columns: ColumnDef[] }
  | { key: string; label: string; type: "rich" }
  | { key: string; label: string; type: "repeat"; itemFields: FieldDef[] }

type DocumentValues = Record<string, unknown>
```

The same model must exist in Go domain types and in docgen payload types.

## Runtime Endpoints to Add

The plan assumes these v1 endpoints:

- `GET /api/v1/document-types`
- `GET /api/v1/document-types/{typeKey}`
- `GET /api/v1/document-types/{typeKey}/bundle`
- `POST /api/v1/documents`
- `GET /api/v1/documents/{documentId}/editor-bundle`
- `PUT /api/v1/documents/{documentId}/content`
- `POST /api/v1/documents/{documentId}/export/docx`

The old profile-centric bundle endpoints may remain temporarily, but the active frontend flow must move to these runtime endpoints by the end of this plan.

---

### Task 1: ADR and API Contract Freeze

**Files:**
- Create: `docs/adr/0020-schema-runtime-document-platform.md`
- Modify: `api/openapi/v1/openapi.yaml`
- Test: `tests/contract/api_contract_smoke_test.go`

- [ ] **Step 1: Write the failing contract test for the new runtime endpoints**

```go
func TestOpenAPIContainsSchemaRuntimeEndpoints(t *testing.T) {
	data, err := os.ReadFile("api/openapi/v1/openapi.yaml")
	if err != nil {
		t.Fatalf("read openapi: %v", err)
	}

	required := []string{
		"/document-types/{typeKey}/bundle:",
		"/documents/{documentId}/editor-bundle:",
		"/documents/{documentId}/content:",
		"/documents/{documentId}/export/docx:",
	}

	for _, needle := range required {
		if !strings.Contains(string(data), needle) {
			t.Fatalf("missing path %s", needle)
		}
	}
}
```

- [ ] **Step 2: Run the contract test to verify it fails**

Run: `go test ./tests/contract -run OpenAPIContainsSchemaRuntimeEndpoints -count=1`

Expected: FAIL because the new runtime paths do not exist yet.

- [ ] **Step 3: Write ADR 0020 and update OpenAPI with the runtime surface**

```md
# ADR-0020 Schema Runtime Document Platform

## Decision

Adopt a schema-driven document runtime inside the existing `documents` module.

## Consequences

- `document_profiles` stop being the active editor contract
- schema + values become the canonical editor and export contract
- admin schema authoring is deferred to a later phase
```

```yaml
/document-types/{typeKey}/bundle:
  get:
    summary: Load document type metadata, active schema, and governance
/documents/{documentId}/editor-bundle:
  get:
    summary: Load document, versions, active schema, values, and collaboration state
/documents/{documentId}/content:
  put:
    summary: Replace the current draft content values for a document
/documents/{documentId}/export/docx:
  post:
    summary: Generate a DOCX from the active schema runtime payload
```

- [ ] **Step 4: Re-run the contract test**

Run: `go test ./tests/contract -run OpenAPIContainsSchemaRuntimeEndpoints -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add docs/adr/0020-schema-runtime-document-platform.md api/openapi/v1/openapi.yaml tests/contract/api_contract_smoke_test.go
git commit -m "docs(runtime): define schema runtime api and adr"
```

### Task 2: Add Runtime Persistence Tables and Seed Data

**Files:**
- Create: `migrations/0048_init_document_type_runtime.sql`
- Create: `migrations/0049_grant_document_type_runtime_privileges.sql`
- Create: `migrations/0050_seed_document_type_runtime.sql`
- Test: `tests/unit/documents_postgres_repository_test.go`

- [ ] **Step 1: Write the failing repository test for schema runtime persistence**

```go
func TestPostgresRepository_SaveDocumentTypeSchemaRuntime(t *testing.T) {
	repo := newPostgresRepositoryForTest(t)
	ctx := context.Background()

	item := domain.DocumentTypeDefinition{
		Key:  "po",
		Name: "Procedimento Operacional",
		ActiveVersion: 1,
		Schema: domain.DocumentTypeSchema{
			Sections: []domain.SectionDef{
				{Key: "identificacao", Num: "1", Title: "Identificacao"},
			},
		},
	}

	if err := repo.UpsertDocumentTypeDefinition(ctx, item); err != nil {
		t.Fatalf("upsert type: %v", err)
	}
}
```

- [ ] **Step 2: Run the repository test to verify it fails**

Run: `go test ./tests/unit -run PostgresRepository_SaveDocumentTypeSchemaRuntime -count=1`

Expected: FAIL because the tables and repository methods do not exist yet.

- [ ] **Step 3: Add additive runtime tables and seed the initial document types**

```sql
CREATE TABLE IF NOT EXISTS metaldocs.document_types (
  type_key TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  family_key TEXT NOT NULL DEFAULT '',
  active_version INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS metaldocs.document_type_schema_versions (
  type_key TEXT NOT NULL REFERENCES metaldocs.document_types(type_key),
  version INTEGER NOT NULL,
  schema_json JSONB NOT NULL,
  governance_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (type_key, version)
);

ALTER TABLE metaldocs.documents
  ADD COLUMN IF NOT EXISTS document_type_key TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS document_type_version INTEGER NOT NULL DEFAULT 1;

ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS values_json JSONB NOT NULL DEFAULT '{}'::jsonb;
```

```sql
INSERT INTO metaldocs.document_types (type_key, name, description, family_key, active_version)
VALUES ('po', 'Procedimento Operacional', 'Runtime type', 'procedure', 1)
ON CONFLICT (type_key) DO UPDATE SET active_version = EXCLUDED.active_version;
```

- [ ] **Step 4: Re-run the repository test**

Run: `go test ./tests/unit -run PostgresRepository_SaveDocumentTypeSchemaRuntime -count=1`

Expected: FAIL with missing repository method or type, which is correct after migrations land but before Go code.

- [ ] **Step 5: Commit**

```bash
git add migrations/0048_init_document_type_runtime.sql migrations/0049_grant_document_type_runtime_privileges.sql migrations/0050_seed_document_type_runtime.sql tests/unit/documents_postgres_repository_test.go
git commit -m "feat(runtime): add schema runtime persistence tables"
```

### Task 3: Introduce Canonical Schema Runtime Types in Go Domain

**Files:**
- Create: `internal/modules/documents/domain/schema_runtime.go`
- Create: `internal/modules/documents/domain/schema_runtime_errors.go`
- Modify: `internal/modules/documents/domain/model.go`
- Modify: `internal/modules/documents/domain/port.go`
- Test: `tests/unit/documents_service_test.go`

- [ ] **Step 1: Write the failing domain validation test**

```go
func TestValidateDocumentTypeSchema_RejectsUnknownFieldType(t *testing.T) {
	schema := domain.DocumentTypeSchema{
		Sections: []domain.SectionDef{
			{
				Key: "s1",
				Num: "1",
				Title: "Section 1",
				Fields: []domain.FieldDef{
					{Key: "x", Label: "X", Type: "unknown"},
				},
			},
		},
	}

	err := domain.ValidateDocumentTypeSchema(schema)
	if !errors.Is(err, domain.ErrDocumentSchemaInvalidField) {
		t.Fatalf("expected schema field error, got %v", err)
	}
}
```

- [ ] **Step 2: Run the domain test to verify it fails**

Run: `go test ./tests/unit -run ValidateDocumentTypeSchema_RejectsUnknownFieldType -count=1`

Expected: FAIL because the runtime schema types do not exist yet.

- [ ] **Step 3: Add canonical runtime types and validation**

```go
type DocumentTypeSchema struct {
	Sections []SectionDef `json:"sections"`
}

type SectionDef struct {
	Key    string     `json:"key"`
	Num    string     `json:"num"`
	Title  string     `json:"title"`
	Color  string     `json:"color,omitempty"`
	Fields []FieldDef `json:"fields"`
}

type FieldDef struct {
	Key        string     `json:"key"`
	Label      string     `json:"label"`
	Type       string     `json:"type"`
	Options    []string   `json:"options,omitempty"`
	Columns    []FieldDef `json:"columns,omitempty"`
	ItemFields []FieldDef `json:"itemFields,omitempty"`
}
```

```go
var allowedFieldTypes = map[string]struct{}{
	"text": {}, "textarea": {}, "number": {}, "date": {}, "select": {},
	"checkbox": {}, "table": {}, "rich": {}, "repeat": {},
}
```

- [ ] **Step 4: Re-run the domain test**

Run: `go test ./tests/unit -run ValidateDocumentTypeSchema_RejectsUnknownFieldType -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/domain/schema_runtime.go internal/modules/documents/domain/schema_runtime_errors.go internal/modules/documents/domain/model.go internal/modules/documents/domain/port.go tests/unit/documents_service_test.go
git commit -m "feat(runtime): add canonical schema runtime domain types"
```

### Task 4: Implement Repository and Application Services for Schema + Values

**Files:**
- Create: `internal/modules/documents/application/service_schema_runtime.go`
- Create: `internal/modules/documents/application/service_document_runtime.go`
- Create: `internal/modules/documents/application/service_runtime_validation.go`
- Modify: `internal/modules/documents/infrastructure/postgres/repository.go`
- Modify: `internal/modules/documents/infrastructure/memory/repository.go`
- Test: `tests/unit/documents_service_test.go`
- Test: `tests/unit/documents_postgres_repository_test.go`

- [ ] **Step 1: Write the failing service test for saving runtime values**

```go
func TestService_SaveDocumentValues_UpdatesDraftInPlace(t *testing.T) {
	repo := memoryrepo.NewRepository()
	service := application.NewService(repo, nil, nil)
	ctx := context.Background()

	doc := seedRuntimeDocument(t, repo, domain.StatusDraft)
	values := map[string]any{"objetivo": "Novo texto"}

	version, err := service.SaveDocumentValuesAuthorized(ctx, domain.SaveDocumentValuesCommand{
		DocumentID: doc.ID,
		Values: values,
		TraceID: "trace-runtime-save",
	})
	if err != nil {
		t.Fatalf("save values: %v", err)
	}
	if version.Number != 1 {
		t.Fatalf("expected in-place draft update, got version %d", version.Number)
	}
}
```

- [ ] **Step 2: Run the service test to verify it fails**

Run: `go test ./tests/unit -run SaveDocumentValues_UpdatesDraftInPlace -count=1`

Expected: FAIL because the save runtime service does not exist yet.

- [ ] **Step 3: Implement repository methods and draft/non-draft save behavior**

```go
type SaveDocumentValuesCommand struct {
	DocumentID string
	Values     map[string]any
	TraceID    string
}
```

```go
func (s *Service) SaveDocumentValuesAuthorized(ctx context.Context, cmd domain.SaveDocumentValuesCommand) (domain.Version, error) {
	doc, err := s.repo.GetDocument(ctx, cmd.DocumentID)
	if err != nil {
		return domain.Version{}, err
	}

	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentEdit)
	if err != nil || !allowed {
		return domain.Version{}, domain.ErrDocumentNotFound
	}

	if doc.Status == domain.StatusDraft {
		return s.updateDraftVersionValues(ctx, doc, cmd)
	}

	return s.addVersionFromRuntimeValues(ctx, doc, cmd)
}
```

- [ ] **Step 4: Re-run the unit tests for service and repository**

Run: `go test ./tests/unit -run "SaveDocumentValues_UpdatesDraftInPlace|PostgresRepository_SaveDocumentTypeSchemaRuntime" -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/application/service_schema_runtime.go internal/modules/documents/application/service_document_runtime.go internal/modules/documents/application/service_runtime_validation.go internal/modules/documents/infrastructure/postgres/repository.go internal/modules/documents/infrastructure/memory/repository.go tests/unit/documents_service_test.go tests/unit/documents_postgres_repository_test.go
git commit -m "feat(runtime): persist schemas and document values"
```

### Task 5: Add Runtime HTTP Endpoints, Auth, Audit, and Export Proxy

**Files:**
- Create: `internal/modules/documents/delivery/http/handler_runtime.go`
- Create: `internal/platform/config/docgen.go`
- Create: `internal/platform/render/docgen/client.go`
- Create: `internal/platform/render/docgen/types.go`
- Modify: `internal/modules/documents/delivery/http/handler.go`
- Modify: `internal/platform/bootstrap/api.go`
- Test: `tests/unit/documents_http_handler_test.go`

- [ ] **Step 1: Write the failing HTTP handler test**

```go
func TestDocumentsHandler_PutRuntimeContent(t *testing.T) {
	handler := newDocumentsHandlerForTest(t)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/documents/doc-1/content", strings.NewReader(`{"values":{"objetivo":"Texto"}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run the handler test to verify it fails**

Run: `go test ./tests/unit -run DocumentsHandler_PutRuntimeContent -count=1`

Expected: FAIL because the endpoint is not registered yet.

- [ ] **Step 3: Add handler routes and docgen proxy client**

```go
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/document-types", h.handleDocumentTypes)
	mux.HandleFunc("/api/v1/document-types/", h.handleDocumentTypeSubRoutes)
	mux.HandleFunc("/api/v1/documents/", h.handleDocumentSubRoutes)
}
```

```go
type Client struct {
	baseURL string
	http    *http.Client
}

func (c *Client) Generate(ctx context.Context, payload RenderPayload) ([]byte, error) {
	// post schema + values to apps/docgen
}
```

- [ ] **Step 4: Re-run the handler test**

Run: `go test ./tests/unit -run DocumentsHandler_PutRuntimeContent -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/delivery/http/handler_runtime.go internal/modules/documents/delivery/http/handler.go internal/platform/config/docgen.go internal/platform/render/docgen/client.go internal/platform/render/docgen/types.go internal/platform/bootstrap/api.go tests/unit/documents_http_handler_test.go
git commit -m "feat(runtime): expose schema runtime document endpoints"
```

### Task 6: Rebuild `apps/docgen` Around Schema + Values

**Files:**
- Create: `apps/docgen/src/runtime/types.ts`
- Create: `apps/docgen/src/runtime/blocks.ts`
- Create: `apps/docgen/src/runtime/renderField.ts`
- Create: `apps/docgen/src/runtime/renderSection.ts`
- Modify: `apps/docgen/src/generate.ts`
- Modify: `apps/docgen/src/index.ts`
- Modify: `apps/docgen/scripts/sample-payload.json`
- Modify: `apps/docgen/scripts/harness.ps1`

- [ ] **Step 1: Write the failing harness payload against the new runtime contract**

```json
{
  "document": {
    "documentId": "doc-1",
    "title": "Procedimento Operacional"
  },
  "schema": {
    "sections": [
      {
        "key": "s1",
        "num": "1",
        "title": "Identificacao",
        "fields": [
          { "key": "objetivo", "label": "Objetivo", "type": "textarea" }
        ]
      }
    ]
  },
  "values": {
    "objetivo": "Texto de teste"
  }
}
```

- [ ] **Step 2: Run the minimal harness**

Run: `powershell -ExecutionPolicy Bypass -File apps/docgen/scripts/harness.ps1`

Expected: FAIL because `generateDocx` still expects the old payload shape.

- [ ] **Step 3: Implement generic field and rich-block rendering**

```ts
export interface RenderPayload {
  document: { documentId: string; title: string };
  schema: DocumentTypeSchema;
  values: Record<string, unknown>;
}

export function renderField(field: FieldDef, value: unknown): Paragraph[] {
  switch (field.type) {
    case "text":
    case "textarea":
      return renderTextField(field, value);
    case "table":
      return renderTableField(field, value);
    case "rich":
      return renderRichField(field, value);
    case "repeat":
      return renderRepeatField(field, value);
    default:
      return [];
  }
}
```

- [ ] **Step 4: Re-run the harness**

Run: `powershell -ExecutionPolicy Bypass -File apps/docgen/scripts/harness.ps1`

Expected: PASS with `tsc --noEmit`, `node dist/index.js`, and a non-zero `.docx` response.

- [ ] **Step 5: Commit**

```bash
git add apps/docgen/src/runtime/types.ts apps/docgen/src/runtime/blocks.ts apps/docgen/src/runtime/renderField.ts apps/docgen/src/runtime/renderSection.ts apps/docgen/src/generate.ts apps/docgen/src/index.ts apps/docgen/scripts/sample-payload.json apps/docgen/scripts/harness.ps1
git commit -m "feat(docgen): render schema runtime payloads"
```

### Task 7: Add Frontend Runtime Types, API Client, and Store Integration

**Files:**
- Create: `frontend/apps/web/src/features/documents/runtime/schemaRuntimeTypes.ts`
- Create: `frontend/apps/web/src/features/documents/runtime/schemaRuntimeAdapters.ts`
- Create: `frontend/apps/web/src/features/documents/runtime/useSchemaDocumentEditor.ts`
- Modify: `frontend/apps/web/src/api/documents.ts`
- Modify: `frontend/apps/web/src/store/documents.store.ts`
- Modify: `frontend/apps/web/src/components/content-builder/contentSchemaTypes.ts`
- Test: `frontend/apps/web/src/components/content-builder/contentSchemaTypes.ts` via TypeScript build

- [ ] **Step 1: Replace the shallow schema type definitions with canonical runtime types**

```ts
export interface DocumentTypeSchema {
  sections: SectionDef[];
}

export interface SectionDef {
  key: string;
  num: string;
  title: string;
  color?: string;
  fields: FieldDef[];
}
```

- [ ] **Step 2: Run the frontend build to surface breakage**

Run: `cd frontend/apps/web; npm run build`

Expected: FAIL because the editor and preview still depend on the old content schema shape.

- [ ] **Step 3: Add runtime API methods and editor hook**

```ts
export async function fetchDocumentTypeBundle(typeKey: string) {
  return client.get<DocumentTypeBundleResponse>(`/document-types/${typeKey}/bundle`);
}

export async function saveDocumentContent(documentId: string, values: Record<string, unknown>) {
  return client.put<DocumentEditorBundleResponse>(`/documents/${documentId}/content`, { values });
}
```

- [ ] **Step 4: Re-run the frontend build**

Run: `cd frontend/apps/web; npm run build`

Expected: FAIL in rendering components only, with the type layer compiling cleanly.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/runtime/schemaRuntimeTypes.ts frontend/apps/web/src/features/documents/runtime/schemaRuntimeAdapters.ts frontend/apps/web/src/features/documents/runtime/useSchemaDocumentEditor.ts frontend/apps/web/src/api/documents.ts frontend/apps/web/src/store/documents.store.ts frontend/apps/web/src/components/content-builder/contentSchemaTypes.ts
git commit -m "feat(frontend-runtime): add schema runtime client and state"
```

### Task 8: Build `DynamicEditor` and `DynamicPreview`

**Files:**
- Create: `frontend/apps/web/src/features/documents/runtime/DynamicEditor.tsx`
- Create: `frontend/apps/web/src/features/documents/runtime/DynamicPreview.tsx`
- Create: `frontend/apps/web/src/features/documents/runtime/fields/ScalarField.tsx`
- Create: `frontend/apps/web/src/features/documents/runtime/fields/TableField.tsx`
- Create: `frontend/apps/web/src/features/documents/runtime/fields/RepeatField.tsx`
- Create: `frontend/apps/web/src/features/documents/runtime/fields/RichField.tsx`
- Create: `frontend/apps/web/src/features/documents/runtime/DynamicEditor.module.css`
- Create: `frontend/apps/web/src/features/documents/runtime/RichField.module.css`
- Modify: `frontend/apps/web/package.json`
- Modify: `frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx`
- Modify: `frontend/apps/web/src/components/content-builder/preview/DocumentPreviewRenderer.tsx`
- Modify: `frontend/apps/web/src/components/content-builder/preview/templates/templateRegistry.ts`

- [ ] **Step 1: Add Tiptap dependencies**

```json
{
  "dependencies": {
    "@tiptap/react": "^2.11.0",
    "@tiptap/starter-kit": "^2.11.0",
    "@tiptap/extension-image": "^2.11.0",
    "@tiptap/extension-table": "^2.11.0",
    "@tiptap/extension-table-row": "^2.11.0",
    "@tiptap/extension-table-cell": "^2.11.0",
    "@tiptap/extension-table-header": "^2.11.0",
    "@tiptap/extension-text-style": "^2.11.0",
    "@tiptap/extension-color": "^2.11.0"
  }
}
```

- [ ] **Step 2: Run install and the frontend build**

Run: `cd frontend/apps/web; npm install`

Run: `cd frontend/apps/web; npm run build`

Expected: FAIL because `DynamicEditor` and `DynamicPreview` still do not exist.

- [ ] **Step 3: Implement runtime field dispatch and switch the content builder**

```tsx
export function DynamicEditor({ schema, values, onChange }: Props) {
  return (
    <>
      {schema.sections.map((section) => (
        <section key={section.key}>
          <h2>{section.num}. {section.title}</h2>
          {section.fields.map((field) => (
            <FieldRenderer
              key={field.key}
              field={field}
              value={values[field.key]}
              onChange={(next) => onChange(field.key, next)}
            />
          ))}
        </section>
      ))}
    </>
  );
}
```

- [ ] **Step 4: Re-run the frontend build**

Run: `cd frontend/apps/web; npm run build`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/package.json frontend/apps/web/src/features/documents/runtime/DynamicEditor.tsx frontend/apps/web/src/features/documents/runtime/DynamicPreview.tsx frontend/apps/web/src/features/documents/runtime/fields/ScalarField.tsx frontend/apps/web/src/features/documents/runtime/fields/TableField.tsx frontend/apps/web/src/features/documents/runtime/fields/RepeatField.tsx frontend/apps/web/src/features/documents/runtime/fields/RichField.tsx frontend/apps/web/src/features/documents/runtime/DynamicEditor.module.css frontend/apps/web/src/features/documents/runtime/RichField.module.css frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx frontend/apps/web/src/components/content-builder/preview/DocumentPreviewRenderer.tsx frontend/apps/web/src/components/content-builder/preview/templates/templateRegistry.ts
git commit -m "feat(frontend-runtime): add dynamic editor and preview"
```

### Task 9: Cut Over Export, Seed Types, and Remove Old Active Flow Assumptions

**Files:**
- Modify: `internal/modules/documents/application/service_profile_bundle.go`
- Modify: `internal/modules/documents/application/service_editor_bundle.go`
- Modify: `internal/modules/documents/application/service_content_docx.go`
- Modify: `internal/modules/documents/delivery/http/handler_content.go`
- Modify: `frontend/apps/web/src/components/content-builder/contentBuilderUtils.ts`
- Modify: `frontend/apps/web/src/components/content-builder/preview/PreviewPanel.tsx`
- Modify: `apps/docgen/scripts/sample-payload.json`
- Test: `tests/unit/documents_service_test.go`
- Test: `tests/unit/documents_http_handler_test.go`

- [ ] **Step 1: Write the failing export integration test**

```go
func TestService_ExportDocxUsesSchemaRuntimePayload(t *testing.T) {
	service, docgen := newRuntimeServiceWithDocgenStub(t)
	ctx := context.Background()

	_, err := service.ExportDocumentDOCX(ctx, "doc-1", "trace-export")
	if err != nil {
		t.Fatalf("export docx: %v", err)
	}

	if !docgen.LastPayload.ContainsSchema {
		t.Fatalf("expected runtime schema payload")
	}
}
```

- [ ] **Step 2: Run the export test to verify it fails**

Run: `go test ./tests/unit -run ExportDocxUsesSchemaRuntimePayload -count=1`

Expected: FAIL because export still routes through legacy assumptions.

- [ ] **Step 3: Cut export and active editor flows to runtime**

```go
payload := docgen.RenderPayload{
	Document: toRenderDocument(doc, version),
	Schema:   runtimeSchema,
	Values:   version.ValuesJSON,
}
```

```ts
export async function exportDocumentDocx(documentId: string) {
  return client.postBlob(`/documents/${documentId}/export/docx`, {});
}
```

- [ ] **Step 4: Run the targeted tests and the docgen harness**

Run: `go test ./tests/unit -run "ExportDocxUsesSchemaRuntimePayload|DocumentsHandler_PutRuntimeContent|SaveDocumentValues_UpdatesDraftInPlace" -count=1`

Run: `powershell -ExecutionPolicy Bypass -File apps/docgen/scripts/harness.ps1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/application/service_profile_bundle.go internal/modules/documents/application/service_editor_bundle.go internal/modules/documents/application/service_content_docx.go internal/modules/documents/delivery/http/handler_content.go frontend/apps/web/src/components/content-builder/contentBuilderUtils.ts frontend/apps/web/src/components/content-builder/preview/PreviewPanel.tsx apps/docgen/scripts/sample-payload.json tests/unit/documents_service_test.go tests/unit/documents_http_handler_test.go
git commit -m "refactor(runtime): cut editor and export flows to schema runtime"
```

### Task 10: Final Verification, Todo Update, and Cutover Notes

**Files:**
- Modify: `tasks/todo.md`
- Modify: `tasks/lessons.md`

- [ ] **Step 1: Run the full backend test suite**

Run: `go test ./...`

Expected: PASS

- [ ] **Step 2: Run the frontend build**

Run: `cd frontend/apps/web; npm run build`

Expected: PASS

- [ ] **Step 3: Run the docgen harness**

Run: `powershell -ExecutionPolicy Bypass -File apps/docgen/scripts/harness.ps1`

Expected: PASS

- [ ] **Step 4: Update task tracking and record one migration lesson**

```md
## Lesson CD - Canonical schema must lead all runtimes
Date: 2026-03-31 | Trigger: correction
Wrong:   Letting frontend, backend, and docgen evolve their own schema shapes independently
Correct: Define one canonical schema runtime contract in the domain layer and project all runtimes from it
Rule:    A schema-driven platform fails if editor, persistence, preview, and export do not share one source-of-truth contract.
Layer:   process
```

- [ ] **Step 5: Commit**

```bash
git add tasks/todo.md tasks/lessons.md
git commit -m "chore(runtime): record schema runtime migration completion"
```

## Self-Review

### Spec coverage

- Product intent and non-goal of not building a Word clone: covered by Tasks 3, 6, 8, and 9 via the constrained field model and runtime editor.
- V1 schemas via seed/manual entry, not admin UI: covered by Tasks 1 and 2.
- Canonical schema contract across backend, frontend, and docgen: covered by Tasks 3, 6, and 7.
- Reuse of existing workflow, permissions, and module boundaries: covered by Tasks 4, 5, and 9.
- Rich field support for text, image, table, and lists: covered by Tasks 6 and 8.
- Data is dev-only and storage can be redesigned: reflected in additive new runtime tables in Task 2 and runtime cutover in Task 9.

### Placeholder scan

- No `TODO`, `TBD`, or “implement later” placeholders remain.
- Every task includes exact files, concrete commands, and a code or payload snippet.
- No task references undefined path names without creating them earlier.

### Type consistency

- The plan consistently uses `DocumentTypeSchema`, `SectionDef`, `FieldDef`, and `DocumentValues`.
- Backend command name is consistently `SaveDocumentValuesCommand`.
- Frontend runtime naming consistently uses `DynamicEditor`, `DynamicPreview`, and `schemaRuntimeTypes`.
