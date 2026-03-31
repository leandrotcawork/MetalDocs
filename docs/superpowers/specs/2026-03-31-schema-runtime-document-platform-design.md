# Schema Runtime Document Platform Design

Date: 2026-03-31
Status: Draft for user review
Scope: Replace the current document-type-specific content architecture with a schema-driven document platform.

## Problem

MetalDocs is no longer solving only a PO editing and export problem. The real product direction is a document platform where different document types are defined by schema and executed by a shared runtime.

The current path is too type-specific:

- content rules are still partially shaped around fixed document types
- editor behavior depends on document-specific assumptions
- docgen is evolving from PO-centric logic instead of from a generic schema contract
- extending the system to new document types would require repeated code changes

Continuing the current PO-focused migration would create avoidable rework. The architectural center must move from hardcoded document types to schema-defined document types.

## Product Intent

This is not a browser-based Word clone.

The platform should allow more freedom than the current structured forms, but it remains schema-guided:

- some sections are always structured and standardized
- some sections are editorial zones with controlled freedom
- users can write rich text, add images, tables, and lists where the schema allows it
- users do not freely design full page layout, pagination, or document composition
- preview and `.docx` generation remain runtime-owned, not user-owned

The target product is hybrid:

- structured-first for repeatable corporate sections
- editorial where needed for richer sections

## Goals

- Make document type a data definition, not a hardcoded implementation.
- Allow new document types to be introduced in v1 by seed or manual schema registration.
- Use one runtime contract for editor, preview, validation, and docgen.
- Keep workflow, permissions, ownership, approval, and governance concepts where they are already sound.
- Redesign storage and contracts freely, because current document data is dev-only and does not require backward compatibility.

## Non-Goals

- Building a full Word-like browser editor.
- Building the admin UI for schema authoring in v1.
- Preserving backward compatibility with current saved document content.
- Keeping PO as the architectural center of the system.

## Core Decision

Adopt a schema-driven document runtime in two phases.

### Phase 1

- document types and schemas are created by seed, manual JSON, or backend setup
- the platform executes those schemas generically
- editor, preview, validation, and docgen all run from the same schema contract

### Phase 2

- admin users create and edit schemas through a dedicated UI

This splits the problem correctly:

- first build the runtime
- then build the schema-authoring product on top of it

## Canonical Model

The central objects become:

### DocumentType

- defines document-type metadata
- owns the schema
- is stored as data

### Document

- is a concrete instance of a document
- references a `DocumentType`
- stores values that conform to the schema
- participates in workflow, permissions, and versioning

### DocumentSchema

The schema is the executable definition of the document:

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
  | { key: string; label: string; type: 'text' | 'textarea' | 'number' | 'date' | 'select' | 'checkbox' }
  | { key: string; label: string; type: 'table'; columns: ColumnDef[] }
  | { key: string; label: string; type: 'rich' }
  | { key: string; label: string; type: 'repeat'; itemFields: FieldDef[] }
```

### DocumentValues

The document instance stores values only:

```ts
type DocumentValues = Record<string, unknown>
```

The schema defines structure. The document stores data. The runtime interprets both.

## Field Model for V1

V1 supports two families of fields.

### Structured fields

- `text`
- `textarea`
- `number`
- `date`
- `select`
- `checkbox`
- `table`

These are used for highly standardized sections with strong validation and predictable rendering.

### Editorial fields

- `rich`

`rich` is not raw HTML and not free page composition. It is a controlled block-based editorial field that supports:

- rich text
- images
- tables
- lists

For repeatable structures such as steps, items, checklist entries, or repeated records, the schema uses `repeat` with nested fields:

```json
{
  "key": "etapas",
  "type": "repeat",
  "itemFields": [
    { "key": "titulo", "type": "text" },
    { "key": "responsavel", "type": "text" },
    { "key": "corpo", "type": "rich" }
  ]
}
```

This keeps the runtime generic. `etapa` is only one possible repeated structure, not a platform primitive.

## Runtime Architecture

The platform runtime has four responsibilities.

### 1. Schema persistence

The backend stores `DocumentType` definitions and their schemas.

### 2. Validation

The backend validates:

- schema definitions
- document values against the schema

Validation remains backend-owned.

### 3. Runtime rendering

The frontend uses the schema to render:

- `DynamicEditor`
- `DynamicPreview`

These are generic runtime components, not type-specific editors.

### 4. Export

`apps/docgen` receives `schema + values` and renders the final `.docx`.

Docgen should not know PO, INFO, IT, or any other specific document type. It should know:

- sections
- fields
- values
- runtime rendering rules for each field type

## Responsibilities by Layer

### Frontend

- render `DynamicEditor` from schema
- render `DynamicPreview` from schema
- collect user input
- submit values to Go APIs

### Go backend

- store document types and schemas
- store document instances and values
- validate values against schema
- enforce auth, permissions, workflow, and versioning
- assemble runtime payloads for preview and export

### apps/docgen

- receive `schema + values`
- render `.docx` from field definitions and values
- stay stateless and rendering-only

## Reuse vs Replace

### Reuse

The following existing areas are worth preserving:

- workflow and approval concepts
- permissions model
- document ownership and status concepts
- Go module and layer structure where it already follows good boundaries
- technical parts of the current docgen implementation that are already generic enough to reuse

### Replace

The following should be treated as replaceable:

- PO-centered content modeling
- Carbone-shaped content assumptions
- any runtime contract centered on fixed document types
- editor or preview logic tied to one type's internal shape
- document-type-specific API contracts that do not generalize cleanly

### Add

New platform-native layers are required:

- `DocumentType` and schema storage
- schema validator
- values validator
- schema runtime for editor
- schema runtime for preview
- schema runtime for docgen

## Data Compatibility Decision

Current document content is dev-only and may be discarded.

Implications:

- storage can be redesigned cleanly
- APIs can be redesigned cleanly
- old content shapes do not constrain the new platform
- compatibility layers should not be added unless they clearly reduce implementation risk

## Versioning Rule

Versioning remains document-level, not field-level and not `etapa`-level.

An edit to one section, one field, or one repeated item is still an edit to the whole document.

The system should not treat repeated editorial items as independent aggregates.

## Risks

- Overgeneralizing too early can produce a weak schema model that does not match real document needs.
- Mixing admin schema authoring into v1 would expand scope and delay the runtime.
- If editor, preview, and docgen do not share the same schema semantics, the platform will drift quickly.
- Rich and editorial fields are the highest-risk area because they are more expressive and harder to validate consistently.

## Mitigations

- Limit v1 to seed or manual schema registration.
- Keep the field type set intentionally small in v1.
- Use one canonical schema contract across frontend, backend, and docgen.
- Make docgen runtime-driven from the start instead of adding more type-specific rendering branches.

## Testing Strategy

The implementation plan must include:

- backend tests for schema validation
- backend tests for document-values validation
- frontend tests for runtime field rendering by schema
- docgen tests for rendering by field type
- end-to-end tests using at least one structured-heavy type and one editorial-heavy type

## Acceptance Direction

This redesign is successful if:

- a new document type can be introduced in v1 without new type-specific editor code
- the editor renders from schema
- the preview renders from schema
- docgen renders from schema
- structured and editorial sections coexist in the same document cleanly
- the platform still preserves governance, permissions, and workflow rules
