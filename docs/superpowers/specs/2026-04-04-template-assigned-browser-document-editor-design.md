# Template-Assigned Browser Document Editor Design

**Date:** 2026-04-04  
**Status:** Proposed  
**Scope:** MetalDocs document authoring v1, template assignment model, browser editor stack, revision binding, and export boundaries

---

## 1. Objective

Define the simplest viable v1 authoring model for MetalDocs that lets users edit documents directly on a document-like surface inside MetalDocs, while preserving template assignment by document type and document lineage.

This design intentionally narrows scope to the editing experience and defers workflow, review, publishing rules, and template authoring inside MetalDocs.

---

## 2. Product Direction

MetalDocs v1 should use an existing browser-native document editor instead of a custom React-rendered governed canvas.

The chosen direction is:

- `CKEditor 5` as the browser editing engine
- versioned templates stored in MetalDocs
- template assignment by type and optional document-lineage override
- document body stored in editor-native format
- `.docx` and `.pdf` treated as derived outputs

This replaces the previous assumption that v1 should build a custom template DSL plus React canvas renderer for authoring.

---

## 3. Why This Direction

The goal is the easiest implementation that still matches the required user experience:

- open a document in MetalDocs
- see the document itself, not a side form
- edit directly on the rendered document surface
- keep templates assignable by type or by specific document lineage
- export to `.docx` when needed

`CKEditor 5` fits this better than the previous custom-canvas direction because it already supports:

- a full browser editing surface
- locked and editable regions
- template-oriented content composition
- Word import/export capabilities if needed later

This avoids prematurely building a custom layout/rendering engine before the product shape is validated.

---

## 4. Template Model

### 4.1 Canonical template format

For v1, the canonical template format should be web-native editor content, not raw DOCX and not a custom MetalDocs layout DSL.

Each template record should contain:

- `template_key`
- `version`
- `editor = "ckeditor5"`
- `content_format`
- canonical editor-native template body
- metadata such as template name, profile/type applicability, and status

### 4.2 DOCX role

DOCX is not the canonical template format in v1.

DOCX may later be used for:

- bootstrap import of a template
- export of a saved document revision

But the internal source of truth for the template remains the web-native editor representation.

### 4.3 Editable boundaries

Templates define:

- fixed text and static document chrome
- locked regions the user cannot change
- editable regions where the user may type or revise content

This gives the required behavior without building a custom template DSL in v1.

---

## 5. Editing Model

When a user opens a draft document in MetalDocs:

1. MetalDocs resolves the assigned template version
2. MetalDocs loads the saved document revision content
3. The document opens in a single browser editing surface
4. The user edits directly inside the document

The UI should not preserve the current split-pane content-builder shell with sidebar form editing and live PDF preview.

The editor should be the primary authoring experience. PDF remains a later derived view, not the live editing surface.

### 5.1 Content storage

For v1, the authored document body is stored in the editor-native content format rather than in the current schema-driven field tree for the full body.

The revision payload should store at minimum:

- document lineage id
- revision/version number
- template key
- template version
- editor-native document body
- timestamps and author metadata

This is the main simplification that makes v1 tractable.

---

## 6. Template Assignment Rules

Template resolution order for a new draft or revision:

1. document-lineage-specific template override
2. otherwise the type default template

Examples:

- `PO` can use a default template
- `PO-XX-Marketing Guides` can override that default with a different template

The resolved template version is snapshotted into the revision when that revision is created.

If later the default template for `PO` changes:

- historical revisions do not change
- already-created revisions do not change
- only future drafts/revisions use the new template version

This preserves the required history behavior.

---

## 7. Revision Model

Each document lineage has multiple revisions.

For the editing slice covered by this design:

- historical revisions remain preserved
- a newer revision may supersede an older one as the active current version
- only the author sees and edits the draft for now

This design does not define the full review/publish workflow. It only requires that revisions remain versioned and that each revision stays bound to the exact template version it used.

---

## 8. Export Model

For v1:

- MetalDocs is the authoring source of truth
- `.docx` is an export format
- `.pdf` is a generated viewing/review artifact

This means:

- no round-trip DOCX editing back into canonical content in v1
- no live PDF split preview required during authoring
- no DOCX file treated as the canonical stored body

This keeps the platform simple and avoids fidelity traps from making Word the internal source of truth.

---

## 9. Out of Scope For V1

The following are explicitly deferred:

- template creation/editing inside MetalDocs
- custom React document canvas renderer
- custom MetalDocs template DSL
- Word as canonical authoring source
- DOCX round-trip import/edit/save as canonical content
- live PDF preview during editing
- full workflow/review/publish implementation details
- collaborative editing

---

## 10. Alternatives Considered

### 10.1 Custom React canvas plus embedded rich editor

Rejected for v1 because it requires building a custom template/render engine before validating the product shape.

### 10.2 Browser-office or full DOCX-native editor integration

Rejected for v1 because it increases integration weight and pushes the platform toward vendor-owned document logic too early.

### 10.3 Existing HTML/browser editor with template and restricted editing

Chosen because it best matches the desired user experience with the lowest implementation cost.

---

## 11. Acceptance Criteria

This design is successful when:

- a document opens as a single document-like editing surface inside MetalDocs
- the user edits directly inside the document instead of filling side forms
- templates can be assigned by type and overridden by document lineage
- each revision snapshots the exact template version it used
- document body content is stored in editor-native format
- MetalDocs exports `.docx` from saved revisions
- PDF is generated as a derived artifact, not used as the editing surface

---

## 12. Final Recommendation

MetalDocs v1 should adopt a template-assigned browser document editor based on `CKEditor 5`.

The platform should:

- use versioned, web-native templates as the canonical template format
- assign templates by type and optional lineage override
- store document body content in editor-native format
- export `.docx` and `.pdf` as derived outputs

This is the simplest path that matches the intended editing experience without locking the project into a premature custom editor architecture.
