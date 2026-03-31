# Docgen Hybrid Gap-Closure Design

Date: 2026-03-31  
Status: Draft for user review  
Scope: Close remaining gaps on `plan/docgen-hybrid` and reconcile the minimal docgen harness.

## Context

The branch `plan/docgen-hybrid` already implements the majority of the docgen migration (Go projection, docgen service, frontend editor wiring). The remaining work is to finalize rich body persistence and API endpoints, reconcile the minimal docgen harness added to `main`, and update OpenAPI.

This design documents the remaining scope only. It treats `plan/docgen-hybrid` as the baseline and does not rework already-implemented features.

## Goals

- Reconcile the minimal docgen harness (typecheck + start + curl) into `plan/docgen-hybrid`.
- Persist rich step bodies as JSONB (`body_blocks`) in `document_versions`.
- Provide GET/PATCH endpoints for etapa body blocks with immutability enforcement, audit log, and domain event.
- Update OpenAPI with the new endpoints.

## Non-Goals

- Re-implementing existing docgen rendering logic.
- Changing section payloads for 1–4, 6–10.
- Altering existing versioning or approval semantics.

## Decisions

### 1) Harness Reconciliation

Bring the minimal docgen harness from `main` into `plan/docgen-hybrid` without removing the richer docgen implementation:

- Keep `apps/docgen` as-is (sections, render logic).
- Add `apps/docgen/scripts/harness.ps1` and `apps/docgen/scripts/sample-payload.json`.
- Ensure the harness criteria still pass:
  - `tsc --noEmit` passes in `apps/docgen`
  - `node dist/index.js` starts
  - `curl -X POST http://localhost:3001/generate` returns a non-empty `.docx` with correct content type

### 2) Persistence Model

Add `body_blocks` to `document_versions`:

- `body_blocks JSONB DEFAULT '[]'`
- Domain type:

```go
type EtapaBody struct {
    Blocks []json.RawMessage `json:"blocks"`
}
```

### 3) API Contract

Add endpoints for step body blocks:

- `GET /api/v1/documents/{id}/versions/{versionId}/etapas/{stepIndex}/body`
- `PATCH /api/v1/documents/{id}/versions/{versionId}/etapas/{stepIndex}/body`

Request/response:

```json
{ "blocks": [ ... ] }
```

### 4) Immutability, Audit, Events

- If the target version is non-draft, create a new draft version and apply the update there.
- Record an audit entry for every update.
- Publish domain event `DocumentEtapaBodyUpdated` with idempotency key:

```
etapa_body_updated:{documentId}:{versionId}:{stepIndex}
```

### 5) OpenAPI

Document both new endpoints in `api/openapi/v1/openapi.yaml` before implementation changes are merged.

## Testing

- Go unit tests for:
  - body block persistence
  - immutability (draft fork)
  - handler request/response
- `go test ./...` after changes
- Docgen harness script continues to pass

