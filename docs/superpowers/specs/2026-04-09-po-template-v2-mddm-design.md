# PO Template v2 (MDDM) Design

Date: 2026-04-09
Status: Draft for review
Owner: Documents module (MDDM/BlockNote path)

## 1. Goal

Create a real PO template (`po-mddm-canvas`) based on `template-po-v2.docx`, adapted to the MDDM architecture, with:

- locked core scaffold
- editable business content
- repeatable sections for operational data
- rich content area in etapas (including images, lists, and tables)

This design replaces Carbone tokenized semantics in editor content while preserving optional token-like helper hints for operators.

## 2. Source Reference

Reference document:

- `C:\Users\leandro.theodoro.MN-NTB-LEANDROT\Downloads\template-po-v2.docx`

Detected canonical section set:

1. IDENTIFICAÇÃO
2. IDENTIFICAÇÃO DO PROCESSO
3. ENTRADAS E SAÍDAS
4. VISÃO GERAL DO PROCESSO
5. DETALHAMENTO DAS ETAPAS
6. CONTROLE E EXCEÇÕES
7. INDICADORES DE DESEMPENHO
8. DOCUMENTOS E REFERÊNCIAS
9. GLOSSÁRIO
10. HISTÓRICO DE REVISÕES

## 3. Design Decisions (Approved)

### 3.1 Structure control

- Sections `1..6`: locked scaffold (cannot remove or reorder).
- Sections `7..10`: optional (can be removed).
- No section reordering allowed for any section that remains.
- Users can edit text and table/repeatable row content.

### 3.2 Repeatables

- `etapas`: minItems = 1
- `kpis`: minItems = 0
- `referencias`: minItems = 0
- `glossario`: minItems = 0
- `revisoes`: minItems = 0

### 3.3 Etapas requirements

Each etapa item must require:

- `titulo`
- `responsavel`
- one rich editable area

Each etapa item may also include optional fields (`prazo`, `observacoes`, `alertas`).

### 3.4 Fluxograma representation

- Remove `fluxogramaUrl` field from section 4.
- Section 4 supports multiple images in rich content (with optional captions).

### 3.5 Token hints

- Carbone token strings are not used as runtime content.
- Token-like hints may appear as helper text only.

## 4. Alternatives Considered

### A) Fully fixed sections and fixed row counts

Pros:
- strict standardization
- simpler validation

Cons:
- low operational flexibility
- poor fit for evolving processos

### B) Fully free form document

Pros:
- maximum flexibility

Cons:
- weak governance
- inconsistent output across teams

### C) Hybrid governance (selected)

Pros:
- preserves compliance-critical scaffold
- allows real-world operational variation where needed
- aligns with MDDM locked block model and repeatable grammar

Cons:
- moderate schema complexity

## 5. MDDM Block Mapping

## 5.1 Core

- `section` for each numbered section shell
- `fieldGroup` + `field` for label/value structured rows
- `repeatable` + `repeatableItem` for collection-like areas
- `richBlock` for free-form content surfaces
- standard content blocks under rich areas:
  - paragraph
  - heading
  - bulletListItem
  - numberedListItem
  - image
  - quote
  - code
  - divider
  - dataTable (+ row/cell)

## 5.2 Section-specific mapping

- Section 1: identification metadata fields (elaborado/aprovado/datas).
- Section 2: process identity fields (`objetivo`, `escopo`, `responsavel`, `canal`, `participantes`).
- Section 3: two-column process inputs/outputs + related docs/systems rows.
- Section 4: rich description + rich media area for multiple fluxograma images.
- Section 5: repeatable etapas; each item has required structured fields plus rich content area.
- Section 6: control points and exceptions fields (rich-capable when needed).
- Sections 7..10: optional sections with repeatable rows/items.

## 6. Validation Rules

- Template metadata must be `editor = mddm-blocknote`, `content_format = mddm`.
- Locked sections and locked internal scaffold cannot be removed/reordered.
- Optional sections can be removed but not reordered when present.
- Repeatable minItems enforced per approved values.
- Etapa item required field validation:
  - non-empty `titulo`
  - non-empty `responsavel`
  - non-empty rich content tree (at least one meaningful child)

## 7. Export Expectations (DOCX)

- Maintain section numbering and titles in export.
- Preserve table semantics for structured sections.
- Preserve multiple images and captions in section 4 and etapa rich areas.
- Preserve list/table formatting from rich content.
- Preserve deterministic ordering from canonicalized MDDM tree.

## 8. Non-Goals

- Carbone token substitution or runtime token expansion.
- Reintroduction of CKEditor/HTML paths.
- Dynamic section reordering UX for this template version.

## 9. Rollout Plan (Design-Level)

1. Add/refresh PO template seed as MDDM definition/body according to this structure.
2. Ensure default PO binding remains `po-mddm-canvas`.
3. Add schema + service tests for locked scaffold and repeatable rules.
4. Add editor integration/e2e checks for:
   - add/remove rows
   - etapa rich content with images/lists/tables
   - optional section removal for `7..10`.

## 10. Open Questions

None for this design baseline. Future variants can extend with role-based field locks.
