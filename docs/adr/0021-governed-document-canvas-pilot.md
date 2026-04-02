# ADR-0021: Governed Document Canvas Pilot

## Status
Proposed

## Context
`docs/superpowers/specs/2026-04-02-governed-document-canvas-design.md` defines the target governed canvas model. The current repository already has the runtime schema, editor bundle, docgen, and Gotenberg pieces needed to start a narrow pilot.

The pilot must stay deliberately small:
- one `PO` template
- no template admin UI
- no collaborative editing

## Decision
Reuse the existing `GET /documents/{documentId}/editor-bundle` endpoint and extend it with:
- `templateSnapshot`
- `draftToken`

Reuse the existing native save path and extend it with `draftToken`.

Persist template snapshot metadata on `document_versions`.

Treat the MetalDocs rich envelope as the canonical persisted shape for rich fields.

Keep PDF as the final visual truth.

## Consequences
- The current runtime editor becomes transitional and will be bypassed by the pilot canvas.
- `DRAFT` saves become compare-and-set mutations instead of always creating a new version.
- Template management UI is deferred.

## Acceptance test
```bash
rg -n "draftToken|templateSnapshot|DocumentTemplateSnapshotResponse|DocumentTemplateNodeResponse|pilot subset|governed canvas" api/openapi/v1/openapi.yaml
```
