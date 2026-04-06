# ADR-0023: Document Authoring V1 Freeze

## Status
Accepted

## Context
The repository currently contains two competing authoring directions documented over recent iterations: governed canvas and template-assigned browser editor. This caused recurring decision drift, scope reopening, and parallel branch divergence.

The team already approved browser-editor v1 direction in ADR-0022 and subsequent implementation, but we still need one explicit freeze decision to lock v1 behavior, guardrails, and change-control rules before continuing.

## Decision
Freeze v1 document authoring on template-assigned browser editor as the production path:
- single document-like editing surface in browser (CKEditor5-based flow)
- versioned templates assigned by profile default with optional lineage override
- revision-bound template snapshot
- draft editing persisted as browser-editor content source
- DOCX/PDF kept as backend-derived artifacts

Governed-canvas assets remain non-primary/legacy for v1 and must not replace the production editing path without a new ADR.

`docs/superpowers/specs/2026-04-06-document-authoring-v1-freeze.md` is adopted as the operational freeze spec for v1.

## Consequences
- Product and engineering gain one stable decision baseline and stop reopening editor architecture in active implementation branches.
- PRs that conflict with the freeze (editor path, canonical model, export ownership) now require explicit ADR update before merge.
- Existing legacy/pilot paths can remain for compatibility, but roadmap and implementation sequencing must follow the frozen v1 path.
- Master implementation defaults are updated to include the v1 authoring baseline.

## Acceptance test
```bash
go test ./...
cd frontend/apps/web
npm.cmd run build
```
