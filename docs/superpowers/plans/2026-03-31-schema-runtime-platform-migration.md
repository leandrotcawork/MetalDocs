# Schema Runtime Platform Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current profile-centric document flow with a schema-driven runtime for document types, editor, preview, persistence, and `.docx` export.

**Architecture:** Keep the existing `documents` module, workflow, auth, and approval flow, but introduce a new schema-runtime core inside that module. Migrate additively at the database layer, cut over HTTP and frontend runtime flows, and make `apps/docgen` render `schema + values` instead of profile-specific payloads.

**Tech Stack:** Go modular monolith, Postgres JSONB, OpenAPI v1, React + TypeScript + Zustand, Tiptap, Node.js + Express + `docx`

---

## Scope Check

This plan spans backend, frontend, and docgen because all three depend on one canonical schema contract and cannot produce a correct working slice independently.

## File Structure

### Backend

- Create `docs/adr/0020-schema-runtime-document-platform.md` for the architecture decision.
- Create `internal/modules/documents/domain/schema_runtime.go` and `schema_runtime_errors.go` for canonical schema/value types and validation errors.
- Create `internal/modules/documents/application/service_schema_runtime.go`, `service_document_runtime.go`, and `service_runtime_validation.go`.
- Create `internal/modules/documents/delivery/http/handler_runtime.go`.
- Create `internal/platform/config/docgen.go`, `internal/platform/render/docgen/client.go`, and `internal/platform/render/docgen/types.go`.
- Modify `internal/modules/documents/domain/model.go`, `port.go`, `infrastructure/postgres/repository.go`, `infrastructure/memory/repository.go`, `delivery/http/handler.go`, and `internal/platform/bootstrap/api.go`.

### Database

- Create `migrations/0048_init_document_type_runtime.sql`
- Create `migrations/0049_grant_document_type_runtime_privileges.sql`
- Create `migrations/0050_seed_document_type_runtime.sql`

### API

- Modify `api/openapi/v1/openapi.yaml`

### Frontend

- Create `frontend/apps/web/src/features/documents/runtime/schemaRuntimeTypes.ts`
- Create `frontend/apps/web/src/features/documents/runtime/schemaRuntimeAdapters.ts`
- Create `frontend/apps/web/src/features/documents/runtime/useSchemaDocumentEditor.ts`
- Create `frontend/apps/web/src/features/documents/runtime/DynamicEditor.tsx`
- Create `frontend/apps/web/src/features/documents/runtime/DynamicPreview.tsx`
- Create `frontend/apps/web/src/features/documents/runtime/fields/ScalarField.tsx`
- Create `frontend/apps/web/src/features/documents/runtime/fields/TableField.tsx`
- Create `frontend/apps/web/src/features/documents/runtime/fields/RepeatField.tsx`
- Create `frontend/apps/web/src/features/documents/runtime/fields/RichField.tsx`
- Create `frontend/apps/web/src/features/documents/runtime/DynamicEditor.module.css`
- Create `frontend/apps/web/src/features/documents/runtime/RichField.module.css`
- Modify `frontend/apps/web/src/api/documents.ts`, `store/documents.store.ts`, `components/content-builder/contentSchemaTypes.ts`, `components/content-builder/ContentBuilderView.tsx`, `components/content-builder/preview/DocumentPreviewRenderer.tsx`, `components/content-builder/preview/templates/templateRegistry.ts`, and `frontend/apps/web/package.json`.

### Docgen

- Create `apps/docgen/src/runtime/types.ts`, `blocks.ts`, `renderField.ts`, `renderSection.ts`
- Modify `apps/docgen/src/generate.ts`, `src/index.ts`, `scripts/sample-payload.json`, and `scripts/harness.ps1`

### Tests

- Modify `tests/contract/api_contract_smoke_test.go`
- Modify `tests/unit/documents_service_test.go`
- Modify `tests/unit/documents_postgres_repository_test.go`
- Modify `tests/unit/documents_http_handler_test.go`

## Canonical Runtime Contract

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

## Runtime Endpoints

- `GET /api/v1/document-types`
- `GET /api/v1/document-types/{typeKey}`
- `GET /api/v1/document-types/{typeKey}/bundle`
- `POST /api/v1/documents`
- `GET /api/v1/documents/{documentId}/editor-bundle`
- `PUT /api/v1/documents/{documentId}/content`
- `POST /api/v1/documents/{documentId}/export/docx`

---

### Task 1: Freeze the contract with ADR + OpenAPI

**Files:** `docs/adr/0020-schema-runtime-document-platform.md`, `api/openapi/v1/openapi.yaml`, `tests/contract/api_contract_smoke_test.go`

- [ ] Write a failing contract smoke test that asserts the runtime endpoints exist in `openapi.yaml`.
- [ ] Run `go test ./tests/contract -run OpenAPIContainsSchemaRuntimeEndpoints -count=1` and verify it fails.
- [ ] Write ADR-0020 and add the runtime endpoints plus canonical schemas to `api/openapi/v1/openapi.yaml`.
- [ ] Re-run `go test ./tests/contract -run OpenAPIContainsSchemaRuntimeEndpoints -count=1` and verify it passes.
- [ ] Commit with `git commit -m "docs(runtime): define schema runtime api and adr"`.

### Task 2: Add additive persistence for document types and values

**Files:** `migrations/0048_init_document_type_runtime.sql`, `migrations/0049_grant_document_type_runtime_privileges.sql`, `migrations/0050_seed_document_type_runtime.sql`, `tests/unit/documents_postgres_repository_test.go`

- [ ] Write a failing repository test for saving and loading a `DocumentTypeSchema`.
- [ ] Run `go test ./tests/unit -run PostgresRepository_SaveDocumentTypeSchemaRuntime -count=1` and verify it fails.
- [ ] Add runtime tables for `document_types`, `document_type_schema_versions`, and `document_versions.values_json`.
- [ ] Seed PO, IT, and RG in the new schema format.
- [ ] Re-run `go test ./tests/unit -run PostgresRepository_SaveDocumentTypeSchemaRuntime -count=1` and verify the failure moves from migration absence to missing Go implementation.
- [ ] Commit with `git commit -m "feat(runtime): add schema runtime persistence tables"`.

### Task 3: Add canonical Go domain and validation

**Files:** `internal/modules/documents/domain/schema_runtime.go`, `internal/modules/documents/domain/schema_runtime_errors.go`, `internal/modules/documents/domain/model.go`, `internal/modules/documents/domain/port.go`, `tests/unit/documents_service_test.go`

- [ ] Write a failing unit test for `ValidateDocumentTypeSchema` rejecting an unknown field type.
- [ ] Run `go test ./tests/unit -run ValidateDocumentTypeSchema_RejectsUnknownFieldType -count=1` and verify it fails.
- [ ] Add canonical Go types for schema, sections, fields, rich blocks, and document values.
- [ ] Add validation for field type, `table.columns`, and `repeat.itemFields`.
- [ ] Re-run `go test ./tests/unit -run ValidateDocumentTypeSchema_RejectsUnknownFieldType -count=1` and verify it passes.
- [ ] Commit with `git commit -m "feat(runtime): add canonical schema runtime domain types"`.

### Task 4: Implement repository and application runtime services

**Files:** `internal/modules/documents/application/service_schema_runtime.go`, `service_document_runtime.go`, `service_runtime_validation.go`, `internal/modules/documents/infrastructure/postgres/repository.go`, `internal/modules/documents/infrastructure/memory/repository.go`, `tests/unit/documents_service_test.go`, `tests/unit/documents_postgres_repository_test.go`

- [ ] Write a failing service test for saving document values into the active draft version.
- [ ] Run `go test ./tests/unit -run SaveDocumentValues_UpdatesDraftInPlace -count=1` and verify it fails.
- [ ] Implement repository methods for document type definitions, active schema lookup, and JSONB values persistence.
- [ ] Implement application services for type bundles, editor bundles, value validation, and save behavior: draft edits update in place, non-draft edits create a new version.
- [ ] Re-run `go test ./tests/unit -run "SaveDocumentValues_UpdatesDraftInPlace|PostgresRepository_SaveDocumentTypeSchemaRuntime" -count=1` and verify it passes.
- [ ] Commit with `git commit -m "feat(runtime): persist schemas and document values"`.

### Task 5: Expose runtime endpoints and Go-to-docgen export

**Files:** `internal/modules/documents/delivery/http/handler_runtime.go`, `internal/modules/documents/delivery/http/handler.go`, `internal/platform/config/docgen.go`, `internal/platform/render/docgen/client.go`, `internal/platform/render/docgen/types.go`, `internal/platform/bootstrap/api.go`, `tests/unit/documents_http_handler_test.go`

- [ ] Write a failing HTTP test for `PUT /api/v1/documents/{documentId}/content`.
- [ ] Run `go test ./tests/unit -run DocumentsHandler_PutRuntimeContent -count=1` and verify it fails.
- [ ] Register runtime routes, enforce auth via existing middleware and `authn.UserIDFromContext(ctx)`, and proxy export through the new docgen client.
- [ ] Re-run `go test ./tests/unit -run DocumentsHandler_PutRuntimeContent -count=1` and verify it passes.
- [ ] Commit with `git commit -m "feat(runtime): expose schema runtime document endpoints"`.

### Task 6: Rebuild `apps/docgen` for `schema + values`

**Files:** `apps/docgen/src/runtime/types.ts`, `blocks.ts`, `renderField.ts`, `renderSection.ts`, `apps/docgen/src/generate.ts`, `apps/docgen/src/index.ts`, `apps/docgen/scripts/sample-payload.json`, `apps/docgen/scripts/harness.ps1`

- [ ] Replace the sample payload with the runtime contract and run `powershell -ExecutionPolicy Bypass -File apps/docgen/scripts/harness.ps1`; verify it fails on the old payload assumptions.
- [ ] Implement generic field rendering for scalar, table, rich, and repeat fields.
- [ ] Keep rich support limited to text, images, tables, and lists.
- [ ] Re-run `powershell -ExecutionPolicy Bypass -File apps/docgen/scripts/harness.ps1` and verify `tsc --noEmit`, `node dist/index.js`, and the sample `/generate` request all pass.
- [ ] Commit with `git commit -m "feat(docgen): render schema runtime payloads"`.

### Task 7: Add frontend runtime types, API, and store integration

**Files:** `frontend/apps/web/src/features/documents/runtime/schemaRuntimeTypes.ts`, `schemaRuntimeAdapters.ts`, `useSchemaDocumentEditor.ts`, `frontend/apps/web/src/api/documents.ts`, `frontend/apps/web/src/store/documents.store.ts`, `frontend/apps/web/src/components/content-builder/contentSchemaTypes.ts`

- [ ] Replace the current shallow schema types with canonical runtime TS types.
- [ ] Add runtime API methods for type bundle, editor bundle, save values, and export.
- [ ] Add a store-backed editor hook that loads schema + values and emits typed updates.
- [ ] Run `cd frontend/apps/web; npm run build` and verify the breakage is now isolated to rendering components.
- [ ] Commit with `git commit -m "feat(frontend-runtime): add schema runtime client and state"`.

### Task 8: Build `DynamicEditor` and `DynamicPreview`

**Files:** `frontend/apps/web/src/features/documents/runtime/DynamicEditor.tsx`, `DynamicPreview.tsx`, `fields/ScalarField.tsx`, `fields/TableField.tsx`, `fields/RepeatField.tsx`, `fields/RichField.tsx`, `DynamicEditor.module.css`, `RichField.module.css`, `frontend/apps/web/package.json`, `frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx`, `frontend/apps/web/src/components/content-builder/preview/DocumentPreviewRenderer.tsx`, `frontend/apps/web/src/components/content-builder/preview/templates/templateRegistry.ts`

- [ ] Add Tiptap dependencies for rich text, images, tables, lists, text styles, and color.
- [ ] Run `cd frontend/apps/web; npm install` and then `cd frontend/apps/web; npm run build`; verify it fails because runtime components do not exist yet.
- [ ] Implement `DynamicEditor` with field dispatch for scalar, table, repeat, and rich fields.
- [ ] Implement `DynamicPreview` from the same runtime schema and remove the active dependency on the old preview template registry.
- [ ] Re-run `cd frontend/apps/web; npm run build` and verify it passes.
- [ ] Commit with `git commit -m "feat(frontend-runtime): add dynamic editor and preview"`.

### Task 9: Cut over export and the active content-builder flow

**Files:** `internal/modules/documents/application/service_profile_bundle.go`, `service_editor_bundle.go`, `service_content_docx.go`, `internal/modules/documents/delivery/http/handler_content.go`, `frontend/apps/web/src/components/content-builder/contentBuilderUtils.ts`, `frontend/apps/web/src/components/content-builder/preview/PreviewPanel.tsx`, `tests/unit/documents_service_test.go`, `tests/unit/documents_http_handler_test.go`

- [ ] Write a failing unit test proving DOCX export now assembles a runtime payload with `schema + values`.
- [ ] Run `go test ./tests/unit -run ExportDocxUsesSchemaRuntimePayload -count=1` and verify it fails.
- [ ] Cut export and active editor bundle flows to the new runtime services.
- [ ] Re-run `go test ./tests/unit -run "ExportDocxUsesSchemaRuntimePayload|DocumentsHandler_PutRuntimeContent|SaveDocumentValues_UpdatesDraftInPlace" -count=1` and verify it passes.
- [ ] Re-run `powershell -ExecutionPolicy Bypass -File apps/docgen/scripts/harness.ps1` and verify it still passes.
- [ ] Commit with `git commit -m "refactor(runtime): cut editor and export flows to schema runtime"`.

### Task 10: Final verification and migration bookkeeping

**Files:** `tasks/todo.md`, `tasks/lessons.md`

- [ ] Run `go test ./...` and verify it passes.
- [ ] Run `cd frontend/apps/web; npm run build` and verify it passes.
- [ ] Run `powershell -ExecutionPolicy Bypass -File apps/docgen/scripts/harness.ps1` and verify it passes.
- [ ] Update `tasks/todo.md` with the completed migration items.
- [ ] Add one durable lesson to `tasks/lessons.md` about keeping one canonical schema contract across backend, frontend, and docgen.
- [ ] Commit with `git commit -m "chore(runtime): record schema runtime migration completion"`.

## Self-Review

### Spec coverage

- Data-defined document types: Tasks 2, 3, and 4.
- V1 manual schema registration: Tasks 1 and 2.
- One runtime for editor, preview, validation, and docgen: Tasks 3, 4, 6, 7, 8, and 9.
- Hybrid structured + editorial model with rich text, images, tables, and lists: Tasks 3, 6, and 8.
- No Word-in-browser scope creep: Tasks 3, 6, and 8 keep rich support constrained to field-level editorial zones.

### Placeholder scan

- No `TODO`, `TBD`, or “implement later” placeholders remain.
- Every task lists exact files and concrete verification commands.

### Type consistency

- The plan consistently uses `DocumentTypeSchema`, `SectionDef`, `FieldDef`, and `DocumentValues`.
- The save path is consistently named runtime `content` or document `values`, not `etapa body` or profile content.
