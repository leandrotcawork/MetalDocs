# Document Authoring V1 Freeze (MetalDocs)

**Date:** 2026-04-06  
**Status:** FROZEN (approved for implementation continuity)  
**Owner:** Product + Engineering  
**Scope:** Document editing architecture, template model, revision binding, export boundaries, and delivery guardrails for v1

---

## 1. Purpose

Freeze the v1 direction to avoid decision drift, parallel conflicting branches, and repeated architecture resets.

This document is the operational source for "how MetalDocs document authoring must work in v1".

---

## 2. Final Architectural Decision (Frozen)

MetalDocs v1 uses **Template-Assigned Browser Editor** as the production path.

- **Primary editor:** CKEditor 5 (browser-native document surface)
- **Template source of truth:** versioned template stored in MetalDocs (`editor=ckeditor5`, `content_format=html`, `body_html`)
- **Document authoring model:** user edits directly in document surface (not side form containers)
- **Export model:** DOCX/PDF are derived artifacts generated server-side

### Explicitly not the v1 production path

- Custom governed React canvas as main authoring engine
- PDF as live editing surface
- DOCX as canonical persisted authoring format

Governed-canvas assets may remain as legacy/pilot support, but they are not the main v1 authoring workflow.

---

## 3. Non-Negotiables (Do Not Reopen Without New ADR)

1. **A single document-like editor surface** is the main edit experience.
2. **Template assignment is versioned** and resolved by:
   - document-lineage override first
   - type/profile default second
3. **Each revision is bound to an exact template snapshot** (`template_key`, `template_version`, `schema_version` where applicable).
4. **Draft edits update only DRAFT revision state** (in-place for current draft).
5. **Historical revisions are immutable**.
6. **DOCX/PDF generation stays backend-owned**.
7. **Renderer outages map to stable availability errors** (`503`, not generic `500`).
8. **Template/schema seed versions must stay aligned** (no default template pointing to absent schema snapshot).

---

## 4. Canonical V1 Data Model

### 4.1 Template Version

Each template version stores at minimum:

- `template_key`
- `version`
- `profile_code`
- `schema_version`
- `name`
- `editor` (`ckeditor5`)
- `content_format` (`html`)
- `body_html`
- `definition_json` (optional support surface; not the primary browser editing payload)

### 4.2 Document Revision (Draft Authoring)

For browser editor flow, draft content is persisted as editor HTML in the current draft revision.

Required runtime bindings:

- `document_id`
- `version_number`
- `template_key`
- `template_version`
- `content_source=browser_editor`
- `content` (HTML body)

---

## 5. UX Freeze (What User Must See)

When opening a draft document:

1. User opens the document.
2. System loads browser editor bundle with template snapshot + draft body.
3. User edits directly in document surface.
4. Save draft persists same draft revision with stale-write/lock checks.
5. Export DOCX uses backend render path; viewer/reviewer opens PDF as derived artifact.

### Forbidden UX regressions

- Returning to side-field + preview split as the primary editing interaction.
- Forcing users to edit through detached schema form containers for this v1 path.

---

## 6. API/Contract Freeze (V1)

Maintain and stabilize browser-editor contracts:

- `GET /api/v1/documents/{documentId}/browser-editor-bundle`
- `POST /api/v1/documents/{documentId}/content/browser`
- `POST /api/v1/documents/{documentId}/export/docx`
- Template listing/assignment endpoints for selecting template versions per profile/document

All fail-closed outcomes used in runtime must remain represented in OpenAPI.

---

## 7. Operational Guardrails

### 7.1 Dev Runtime

Local flow must assume:

- API service running
- docgen service running (sidecar)
- frontend running

If docgen is unavailable, export must return a stable "renderer unavailable" response path.

### 7.2 Testing Baseline

Before closing any branch touching this flow:

- `go test ./...` passes
- `frontend/apps/web` build passes
- browser editor smoke path passes (manual or automated)
- export DOCX path is verified with docgen up

---

## 8. Deferred (Out of V1 Scope)

The following are postponed and must not expand current v1 scope:

- Full workflow/publish redesign
- Template visual builder inside MetalDocs
- Collaborative real-time editing
- Field-level ACL inside template slots
- Full custom React canvas engine as replacement for browser editor

---

## 9. Branching & Change Control

To avoid losing direction:

1. Architecture changes to this freeze require a new ADR.
2. Any new implementation plan must reference this freeze doc.
3. PRs that contradict Sections 2-6 must be rejected or flagged for ADR first.
4. Keep feature diffs small and vertical (contract + backend + frontend + verification).

---

## 10. Source Traceability

This freeze consolidates and supersedes ambiguity between:

- `docs/superpowers/specs/2026-04-02-governed-document-canvas-design.md`
- `docs/superpowers/specs/2026-04-04-template-assigned-browser-document-editor-design.md`
- `docs/superpowers/plans/2026-04-04-browser-document-editor-v1.md`

Interpretation rule for v1:

- If prior docs conflict, follow this freeze doc.

