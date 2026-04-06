# PO Browser Template Redesign — Design Spec
**Date:** 2026-04-06
**Status:** Approved

---

## Problem

The current `po-default-browser` template renders as plain, unstyled HTML. It has no
document identity header, no enterprise palette, and its Section 5 (Etapas) forces a
rigid field structure that prevents users from describing process steps freely.

---

## Goal

Redesign the PO browser template and establish a **global document design system** that
all future templates will reuse — so that styling, header injection, and editor wiring
are built once and never repeated per template.

---

## Architecture: Three Layers

### Layer 1 — Shared content stylesheet (`document-content.css`)

**File:** `frontend/apps/web/src/styles/document-content.css`

A single CSS file injected into every CKEditor browser editor via the `contentStylesheets`
config option. Styled in the enterprise vinho palette (`tokens.css`). Defines a complete
set of document primitives that any template can use. No template-specific CSS ever.

**Primitives:**

| Class | Purpose |
|---|---|
| `.md-doc-shell` | Root page wrapper — max-width ~820px, centered, white, DM Sans |
| `.md-doc-header` | Locked identity block — dark vinho background, white text, code + revision + title + meta strip |
| `.md-section` | Content section — vinho-muted left border, h2 in vinho-d |
| `.md-field` | Labeled editable slot — strong label + restricted-editing-exception span in vinho-pale box |
| `.md-free-block` | Open rich-text area — vinho-muted left border, vinho-pale background, entire block is one restricted-editing-exception div |
| `.md-table` | Styled table — vinho thead with white text, clean borders, alternating vinho-pale rows |

**Palette tokens used (from `tokens.css`):**

```
--vinho:       #6b1f2a   (header background, section accents)
--vinho-d:     #3e1018   (h2 headings)
--vinho-l:     #8b2e3a   (hover states)
--vinho-pale:  #f9f3f3   (field backgrounds, free block background)
--vinho-muted: #dfc8c8   (borders, left rails)
--text:        #1a0e0e   (body text)
--text-soft:   #483030   (labels, secondary text)
--border:      #e8dede   (table borders)
font-family:   "DM Sans", sans-serif
```

**Injected via CKEditor config** in `BrowserDocumentEditorView` (or the config factory it
uses). This CSS applies to the editor content area only — it does not pollute the app shell.

---

### Layer 2 — Global header injection (`lib.templateHeader.ts`)

**File:** `frontend/apps/web/src/features/documents/browser-editor/lib.templateHeader.ts`

A pure function called once when the browser editor loads, before CKEditor is initialized.
Takes raw template HTML and document bundle metadata. Prepends a fully-built, locked
`.md-doc-header` block. Returns the final HTML handed to CKEditor.

```ts
function interpolateDocumentHeader(templateHtml: string, meta: DocumentBundleMeta): string
```

**Fields populated from the bundle (`BrowserEditorBundle`):**

| Header field | Source | Notes |
|---|---|---|
| Document code | `bundle.Document.DocumentCode` | e.g. `PO-110` |
| Revision | `bundle.Versions[last].Number` formatted as `Rev. 01` | |
| Document title | `bundle.Document.Title` | |
| Tipo | `bundle.Document.DocumentType` | e.g. `Procedimento Operacional` |
| Elaborado por | frontend auth state (current user display name) | `OwnerID` is in the bundle; name resolved from auth context |
| Data de criação | `bundle.Document.CreatedAt` formatted as `dd/mm/yyyy` | |
| Status | `bundle.Document.Status` | e.g. `Rascunho`, `Aprovado` |
| Aprovado por | not in model yet — renders `—` | populated in future when workflow approval is wired |

The entire header block contains no `restricted-editing-exception` — it is fully locked.
The user sees it, scrolls past it, and it exports with the document. They cannot edit it.

**This function is template-agnostic.** It receives `(templateHtml, meta)` with no
knowledge of which template type it is. Every future template gets a correct header
automatically.

---

### Layer 3 — Template HTML (`po-default-browser`)

The template body uses only the shared primitives. No inline styles. No per-template CSS.
The header block is absent — injection adds it at load time.

#### Section 2 — Identificação do Processo
`.md-section` + five `.md-field` slots:
- Objetivo
- Escopo
- Cargo responsável
- Canal / Contexto
- Participantes

#### Section 3 — Entradas e Saídas
`.md-section` + four `.md-field` slots:
- Entradas
- Saídas
- Documentos relacionados
- Sistemas utilizados

#### Section 4 — Visão Geral do Processo
`.md-section` + four `.md-field` slots:
- Descrição do processo
- Ferramenta do fluxograma
- Link do fluxograma
- Diagrama (larger free area — image or text description)

#### Section 5 — Detalhamento das Etapas *(free-form)*
`.md-section` + a locked instruction paragraph + **one model `.md-free-block`**.

The instruction (locked, non-editable):
> *"Descreva cada etapa como uma seção livre. Duplique o bloco abaixo para adicionar mais etapas."*

The model block (entire block is `restricted-editing-exception`):
```
Etapa 1 — [Nome da etapa]          ← H3, user edits freely
[Free prose, bullets, sub-steps, references, anything]
```

User duplicates the block for each additional step. No rigid field structure. Each etapa
can be as long or short, as structured or free as needed — exactly like narrative phase
descriptions (e.g. "3.3.1 FASE DE PRE-PROJETO" prose style from ISO documents).

#### Section 6 — Controle e Exceções
`.md-section` + two `.md-field` slots:
- Pontos de controle
- Exceções e desvios

#### Section 7 — Indicadores de Desempenho
`.md-section` + one `.md-table`:
- Columns: `Indicador / KPI` | `Meta` | `Frequência`
- Pre-seeded with one example row inside `restricted-editing-exception`

#### Section 8 — Documentos e Referências
`.md-section` + one `.md-table`:
- Columns: `Código` | `Título / Descrição` | `Link / Localização`
- Pre-seeded with one example row

#### Section 9 — Glossário
`.md-section` + one `.md-table`:
- Columns: `Termo` | `Definição`
- Pre-seeded with one example row

---

## What changes in the codebase

| File | Change |
|---|---|
| `frontend/apps/web/src/styles/document-content.css` | **New** — shared document design system |
| `frontend/apps/web/src/features/documents/browser-editor/lib.templateHeader.ts` | **New** — global header injection utility |
| `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx` | Wire `contentStylesheets` + call `interpolateDocumentHeader` on load |
| `internal/modules/documents/domain/template.go` | Rewrite `po-default-browser` body using shared primitives |
| `migrations/0058_update_po_browser_template.sql` | Update `body_html` in DB to match new template |

---

## What does NOT change

- The Go backend domain model (`DocumentTemplateVersion`, `DocumentTemplateSnapshot`) — unchanged
- The restricted editing plugin setup — unchanged, same `restricted-editing-exception` pattern
- The CKEditor toolbar — unchanged
- Sections 1 and 10 (Identificação auto-generated, Histórico de Revisões) — auto-generated by docgen on export, not in the browser template body
- All existing tests — the Go/SQL parity test will need updating for the new template body, but the test structure is unchanged

---

## Future templates

When a new template is added (IT, RG, MN, etc.):
1. Write the template HTML using only `.md-section`, `.md-field`, `.md-free-block`, `.md-table`
2. Seed it in Go (`domain/template.go`) and SQL migration
3. Header and styling appear automatically — no new CSS, no new injection code

When the in-system template builder is built:
- The authoring UI exposes the same primitives as building blocks
- The CSS layer is already complete — no changes needed
- The header injection is already global — no changes needed

---

## Reference documents

- `template-po-v2.docx` — structural reference (section numbering, field names, table layouts)
- `PO-05-04 Projeto e Desenvolvimento.pdf` — Etapas style reference (free narrative phases)
- `frontend/apps/web/src/styles/tokens.css` — enterprise palette
- `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.module.css` — existing editor chrome CSS
