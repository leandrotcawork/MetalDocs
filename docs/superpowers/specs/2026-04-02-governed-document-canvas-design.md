# Governed Document Canvas Design

**Date:** 2026-04-02
**Status:** Proposed
**Scope:** MetalDocs document authoring UX, template model, rich content model, renderer contract, and revision snapshot behavior

---

## 1. Objective

Define the target authoring architecture for MetalDocs so users can edit documents in a way that feels document-like and professional, while preserving the platform's metadata-driven, profile-driven, versioned, and auditable foundations.

The goal is to replace the false choice between:

- rigid schema-only form editing that makes writing hard
- uncontrolled full-document WYSIWYG editing that weakens governance

with a governed hybrid model:

- structured fields for controlled data
- rich editable regions for narrative content
- template-defined document canvas rendering for better authoring UX
- Go-owned canonical export pipeline for DOCX/PDF generation

---

## 2. Current Context

The current repository already has the right strategic foundations:

- profile-driven and schema-driven document modeling
- immutable non-draft version history
- backend validation and authorization
- unified Go render payload builder
- `apps/docgen` for `.docx` generation
- Gotenberg for `.pdf` conversion
- PDF-based preview instead of frontend-only fake rendering

Relevant current anchors:

- Schema runtime direction: `docs/adr/0020-schema-runtime-document-platform.md`
- Registry/schema hydration through service layer: `internal/modules/documents/application/service_registry.go`
- Unified docgen payload builder: `internal/modules/documents/application/service_document_runtime.go`
- Native content save/render flow: `internal/modules/documents/application/service_content_native.go`
- Final DOCX renderer: `apps/docgen/src/generate.ts`
- Final PDF conversion: `internal/platform/render/gotenberg/client.go`
- Current runtime editor: `frontend/apps/web/src/features/documents/runtime/DynamicEditor.tsx`
- Current rich editor: `frontend/apps/web/src/features/documents/runtime/fields/RichField.tsx`

The main architectural gap is not export. It is authoring:

- current editing still feels closer to application forms than to editing a governed document
- current rich content representation is mixed
- layout semantics live too heavily in export code instead of in a shared authoring model

---

## 3. Product Decision

MetalDocs should adopt a **Governed Document Canvas** authoring model.

### Summary

Users should edit a document-like canvas that is rendered from:

- a versioned schema snapshot
- a versioned template snapshot
- canonical structured document values

Users may edit only inside template-defined editable regions.

Users may **not**:

- create new top-level sections inside a document
- move blocks around freely
- alter page layout visually
- bypass the template-defined document skeleton

The authoring experience should feel close to writing on the document itself, but the underlying persistence model must remain structured and governed.

### Why this is the right product model

This model preserves:

- writing ergonomics
- controlled document structure
- version reproducibility
- auditability
- export determinism
- template governance

This is the correct middle ground between:

- schema-only forms
- full-document Word-like free editing

---

## 4. Non-Goals

This design explicitly does **not** include:

- freeform desktop-publishing behavior
- drag/drop layout editing by end users
- arbitrary top-level section creation inside documents
- raw HTML templates as the canonical template system
- HTML as canonical rich content persistence
- direct DOCX or PDF editing in the browser
- exact browser-to-PDF visual parity at every keystroke
- admin visual template designer in the first implementation phase

---

## 5. Target Domain Model

The model separates governance, structure, presentation, lineage, and revision state.

### 5.1 Profile

Profile is the platform-level governed entry point, such as `PO`, `IT`, or `RG`.

Profile owns:

- family/type relationship
- workflow defaults
- governance defaults
- taxonomy constraints
- default schema assignment
- default template assignment

Profiles remain the main entry point for document creation.

### 5.2 Schema Version

Schema version defines the structural contract of document content.

Schema owns:

- sections
- field keys
- field kinds
- repeat/table shapes
- required vs optional content
- validation rules

Schema answers:

> What content exists and what shape must it have?

Schema does **not** define exact document-canvas presentation.

### 5.3 Template Version

Template version defines the presentation contract for a compatible schema version.

Template owns:

- page shell composition
- section layout
- visual composition of static and editable regions
- slot placement and binding paths
- static labels, dividers, tables, and document chrome

Template answers:

> How is this schema rendered as an editable document canvas?

For the first implementation, each template version must bind to exactly one schema version.

Compatibility ranges are explicitly out of scope for v1 and may be added only after the one-to-one model is proven in production.

### 5.4 Document Lineage

A document lineage is the stable logical document identity.

Examples:

- one operational procedure lineage
- one logistics instruction lineage
- one marketing process lineage

Lineage owns:

- `document_id`
- profile
- identity metadata
- current workflow state
- default template resolution context for future drafts
- optional lineage-specific template override

Lineage is stable across revisions.

### 5.5 Revision Snapshot

Each revision must snapshot:

- schema version used
- template version used
- canonical content payload
- generated artifact references

Old revisions never resolve "latest template" or "latest schema" implicitly.

This is mandatory for auditability and reproducibility.

---

## 6. Template Resolution Rules

Template resolution for a new draft or revision follows this order:

1. document-lineage-specific template override
2. profile default template

The resolved template version is then snapshotted into the revision context.

For v1, the resolution result must always be a single concrete template version bound to a single concrete schema version.

### Important consequence

If a lineage later changes its assigned template:

- existing revisions remain on their original template snapshot
- only future drafts or revisions may use the new template version
- migration must be explicit, never automatic

This matches the required behavior:

- a general default template for all `PO`
- specific lineages allowed to override the default
- old revisions remaining historically stable

---

## 7. Renderer Contract

The governed document canvas must be rendered from a structured render contract, not from ad hoc forms or raw HTML templates.

### 7.1 Frontend renderer inputs

The frontend document canvas renderer consumes:

- `schemaSnapshot`
- `templateSnapshot`
- `documentValues`
- `editorState`

and returns:

- document-canvas HTML rendered through React components

### 7.2 Template must be structured, not raw HTML

Do not store raw HTML blobs as the canonical template model.

Instead, use a MetalDocs-owned structured template DSL with controlled node types such as:

- `page`
- `stack`
- `columns`
- `section-frame`
- `label`
- `static-text`
- `field-slot`
- `rich-slot`
- `repeat-slot`
- `table-slot`
- `divider`
- `image-slot`
- `metadata-cell`

This allows:

- schema/template compatibility validation
- migration tooling
- deterministic rendering
- reusable canvas components

### 7.3 Slot binding

Editable slots are first-class nodes.

Examples:

- `field-slot(path="identificacao.objetivo")`
- `rich-slot(path="etapas[].descricao")`
- `table-slot(path="kpis")`
- `repeat-slot(path="etapas")`

At render time, the frontend validates:

- path exists in schema snapshot
- slot kind matches schema field kind
- persisted value shape matches expected field shape

This must fail visibly in development and admin modes rather than drifting silently.

### 7.4 Separation of responsibilities

Schema defines:

- what content exists
- what kind of content it is
- how it is validated

Template defines:

- how that content is visually composed
- where editable regions live on the document canvas

This separation is non-negotiable.

---

## 8. Editor Model

### 8.1 Authoring surface

The author edits a document-like canvas, not a detached form.

The page visually resembles the document:

- document header/chrome
- section framing
- predefined tables
- labels and fixed text
- inline editable regions

### 8.2 Allowed editing behavior

Users may:

- fill predefined fields
- type inside predefined rich regions
- add repeat items where schema allows
- edit predefined table cells
- insert allowed content blocks inside rich regions

Users may not:

- create arbitrary top-level sections
- move blocks freely around the page
- change layout placement
- edit outside template-defined regions

### 8.3 Rich editing model

Use rich editing only inside governed `rich-slot` regions.

Allowed rich content should be a controlled set of block/mark capabilities such as:

- paragraph
- heading
- bullet list
- numbered list
- image
- callout
- simple table later
- inline emphasis
- controlled color options
- limited style controls if approved

This provides strong writing flexibility without turning the system into ungoverned HTML.

### 8.4 Recommended editor technology

Use:

- **React** for the outer document canvas and structured slot rendering
- **TipTap / ProseMirror** only for rich editable regions

Do not use:

- DOCX as the browser-native editing surface
- PDF as the browser-native editing surface
- full-document freeform WYSIWYG as the primary authoring model

TipTap is the right embedded rich-region tool because it supports:

- JSON persistence
- React integration
- controlled node/mark sets
- structured extension model

---

## 9. Canonical Content Model

The canonical persistence rule is:

- typed fields stay typed
- rich fields stay structured
- HTML is derived only
- DOCX/PDF are derived only

### 9.1 Canonical values tree

Per revision, persist one canonical content payload keyed by schema structure.

Conceptually:

- scalar fields as scalar values
- table fields as structured table arrays
- repeat fields as structured item arrays
- rich fields as canonical rich JSON values

### 9.2 Rich content source of truth

Canonical rich content must be structured JSON.

Recommended practical choice:

- a MetalDocs-owned rich-content envelope as the stored source of truth for rich slots

Recommended envelope shape:

- `format`
- `version`
- `content`

For v1:

- `format = "metaldocs.rich.tiptap"`
- `version = 1`
- `content = TipTap / ProseMirror JSON`

This keeps the contract owned by MetalDocs while still using TipTap as the current editor implementation.

This is preferred over HTML because it is:

- structured
- versionable
- easier to validate
- easier to render back into the editor
- easier to project into export blocks

### 9.3 HTML usage

HTML may be used only as:

- transient editor output
- clipboard/import/export helper
- read-only derived representation when needed

HTML must not be:

- canonical persisted content
- the primary backend contract
- the export source of truth

### 9.4 Rich validation

Backend validation must enforce:

- allowed node types
- allowed marks
- image policy
- size/depth limits when needed
- per-field capability restrictions if required later

Validation must be schema-aware and profile-aware.

### 9.5 Projection to docgen

Docgen should not be forced to understand arbitrary editor internals forever.

Instead:

1. backend receives canonical rich envelope
2. backend validates it
3. backend projects its `content` into render-oriented rich blocks
4. docgen renders those blocks into `.docx`

This keeps:

- authoring format separate from render format
- export layer deterministic
- editor replacement or evolution possible later

---

## 10. Images and Embedded Assets

Images inserted into rich regions must be governed assets.

Recommended model:

- user uploads image through controlled attachment flow
- image becomes platform-managed immutable asset/blob
- rich content references stable internal asset identity
- rich references include only governed fields such as `asset_id`, `alt`, `caption`, and optional approved variant metadata
- backend resolves binary during render/export

Asset rules for v1:

- each asset has a stable platform identity
- binary payload is immutable once persisted
- rich nodes reference assets by identity, never by arbitrary external URL
- revisions reference the asset identity used at the time of save/export
- asset metadata required for export fidelity, especially `alt` and optional `caption`, must be defined in the contract up front

Do not rely on arbitrary external image URLs as canonical document content.

Reasons:

- portability
- reproducibility
- security
- offline/export reliability

---

## 11. End-to-End Lifecycle

### 11.1 Create

When the user clicks `New Document`:

1. user selects profile/type
2. backend resolves active schema for that profile
3. backend resolves template:
   - document-lineage-specific template if applicable
   - otherwise profile default template
4. system creates document lineage
5. system creates initial draft revision snapshot with:
   - schema version
   - template version
   - initial values
6. editor opens on that revision snapshot

This is a hard rule for the governed canvas flow.

The frontend may keep transient unsaved local UI state after the snapshot is loaded, but it must not invent a client-only document draft lineage before the server creates the initial `DRAFT` revision snapshot.

### 11.2 Edit

Editor loads:

- document lineage metadata
- revision snapshot
- schema snapshot
- template snapshot
- current canonical values

Frontend renders the governed document canvas from those snapshots.

### 11.3 Save draft

On save/autosave:

1. frontend serializes canonical values payload
2. backend validates against schema snapshot
3. backend validates rich JSON policy
4. if current revision is `DRAFT`, update it in place
5. audit/save metadata recorded
6. preview artifacts may be refreshed

### 11.4 Revision creation

When a new revision is required:

1. new revision is created
2. revision snapshots:
   - schema version
   - template version
   - canonical content payload
3. old revisions remain untouched

### 11.5 Template evolution

If template/schema changes later:

- old revisions remain on old snapshots
- existing lineages may explicitly adopt new versions for future drafts
- no automatic rebind of history

### 11.6 Preview

Preview remains generated output:

1. author edits governed canvas
2. save persists canonical content
3. backend projects canonical content into render payload
4. docgen generates DOCX
5. Gotenberg generates PDF
6. frontend preview panel shows generated PDF

This preserves the current strong direction:

- canvas for authoring
- PDF for final visual truth

### 11.7 Export

Export flow:

1. load revision snapshot
2. load schema/template snapshot references
3. build canonical render payload in Go
4. generate DOCX through docgen
5. generate PDF through Gotenberg when needed

---

## 12. Implementation Direction From Current Repo

### 12.1 Keep

Keep:

- profile/schema registry approach
- backend schema hydration and resolution through service layer
- unified Go render payload builder
- docgen as final DOCX renderer
- Gotenberg as final PDF converter
- PDF preview as final user-facing preview
- draft-in-place editing model for `DRAFT`

### 12.2 Replace or evolve

#### Rich persistence

Current issue:

- frontend rich editor emits HTML

Target:

- frontend rich editor emits canonical rich JSON
- backend validates canonical rich JSON
- backend projects rich JSON to docgen render blocks

#### Editor composition

Current issue:

- current runtime editor is schema-driven but still form-like

Target:

- template-driven document canvas renderer
- schema-bound editable slots
- inline structured editing inside the canvas

#### Template model

Current issue:

- template semantics are still implicit or export-only

Target:

- explicit template entity/version
- profile default assignment
- document-lineage override assignment
- revision-level template snapshot

### 12.3 Add

Add:

- versioned template registry in `documents`
- frontend document canvas renderer
- slot-based template renderer
- canonical rich JSON serializer/deserializer
- backend rich JSON validator
- backend projection from rich JSON to docgen render blocks
- explicit revision references to template version and schema version
- explicit rich envelope contract owned by MetalDocs
- explicit governed asset reference contract for rich content

### 12.4 Avoid broad refactor

Do not:

- rewrite the entire documents module before the contract is stable
- build admin visual template tooling first
- over-generalize before validating one profile end to end

---

## 13. Safe Migration Strategy

### Phase 1 - Canonical rich content consolidation

- choose canonical MetalDocs rich envelope
- make frontend rich editor persist the envelope with TipTap content
- add backend validation for the envelope and its TipTap payload
- add backend projection into docgen blocks
- define the governed asset contract for rich content before image-heavy authoring is expanded

### Phase 2 - Template model introduction

- add template entity/version
- add profile default template assignment
- add lineage-specific template override assignment
- add revision template snapshot references
- keep template-to-schema compatibility strictly one-to-one

### Phase 3 - First governed canvas implementation

- implement frontend document canvas renderer for one profile only, preferably `PO`
- keep scope to one schema version and one template version
- validate one rich-slot-heavy document shape end to end
- validate one save/render/export path end to end
- no general admin builder yet
- use controlled template definitions

### Phase 4 - Slot-based editing inside canvas

- mount scalar, repeat, table, and rich editors into template-defined slots
- keep save flow writing canonical payload only

### Phase 5 - End-to-end preview/export stabilization

- ensure save -> Go projection -> docgen -> Gotenberg -> preview is stable
- verify revision reproducibility

### Phase 6 - Expansion and tooling

- expand to more profiles
- only then consider admin tooling for template management

---

## 14. Risks and Anti-Patterns

### 14.1 Two render truths

Risk:

- frontend canvas and export renderer drift apart

Mitigation:

- shared semantic contract for schema/template/content
- PDF remains final output validation

### 14.2 HTML as hidden source of truth

Risk:

- future exports, migrations, and validation become HTML parsing problems

Mitigation:

- canonical structured JSON only

### 14.3 Raw HTML templates

Risk:

- brittle templates, weak validation, poor migrations

Mitigation:

- structured MetalDocs-owned template DSL

### 14.4 Blurred schema/template boundaries

Risk:

- content structure and layout semantics entangle

Mitigation:

- strict responsibility split

### 14.5 Template override explosion

Risk:

- every lineage gets its own template and maintenance collapses

Mitigation:

- profile defaults first
- explicit governance for overrides
- admin visibility into overrides

### 14.6 Uncontrolled author freedom

Risk:

- ad hoc sections and visual drift

Mitigation:

- no user-created top-level sections
- no free movement
- editing only inside predefined slots

### 14.7 Unrealistic fidelity expectations

Risk:

- team tries to make browser canvas perfectly equal DOCX/PDF engine at every moment

Mitigation:

- canvas is authoring-oriented
- generated PDF is final visual truth

### 14.8 Overbuilding tooling too early

Risk:

- visual admin template tooling hardens the wrong model before the contract is proven

Mitigation:

- hand-authored controlled templates first

### 14.9 Revision reproducibility loss

Risk:

- old revisions resolve latest definitions

Mitigation:

- mandatory schema/template snapshotting per revision

### 14.10 Editor-vendor lock-in

Risk:

- editor internals leak too deeply into domain contracts

Mitigation:

- use TipTap JSON as practical authoring format
- define MetalDocs-owned allowed subset and validation policy
- isolate export through backend projection layer

---

## 15. Non-Negotiable Rules

- No user-created top-level sections inside authored documents.
- No free drag/drop layout editing for end users.
- Schema defines content contract; template defines document composition.
- Template activation must validate against schema compatibility.
- For v1, each template version must bind to exactly one schema version.
- Each revision must snapshot schema version and template version.
- Canonical rich content must be structured JSON, not HTML.
- Canonical rich content must be wrapped in a MetalDocs-owned envelope before persistence.
- DOCX and PDF are derived artifacts only.
- Frontend canvas is for authoring; generated PDF is the final visual truth.
- The server must create the initial `DRAFT` revision snapshot before the governed canvas editor opens.
- Document-specific template overrides are explicit governance actions.
- Backend remains authoritative for validation, audit, and export projection.

---

## 16. Acceptance Criteria

This design is considered successfully implemented when:

- users edit inside a document-like canvas instead of a detached form
- authors can write flexible rich content inside governed regions
- top-level document structure remains template-governed
- profile default templates and lineage overrides both work
- old revisions remain bound to their original schema/template snapshots
- canonical rich content is stored as structured JSON
- DOCX export remains deterministic through docgen
- PDF preview remains generated output through Gotenberg
- no critical business rules move into the frontend
- new profiles/templates can be added without hardcoded conditional logic

---

## 17. Final Recommendation

MetalDocs should evolve toward a **Governed Document Canvas** architecture.

This is the best fit for the product because it preserves:

- writing quality
- governance
- auditability
- version stability
- export determinism
- long-term platform scalability

The most important technical decision in this design is not visual polish.
It is the contract:

- versioned schema
- versioned template
- canonical structured content
- explicit revision snapshotting
- separate authoring and export renderers over one semantic model

If those boundaries remain strict, the platform can support document-like authoring without sacrificing the metadata-driven architecture that defines MetalDocs.
