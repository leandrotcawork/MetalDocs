# PO Browser Template — Final Form Design

**Date:** 2026-04-06
**Status:** Approved
**Scope:** DOCX export fidelity, template body field layout, Section 10 addition, bundle token substitution
**References:** `2026-04-06-document-authoring-v1-freeze.md`, `2026-04-06-po-browser-template-redesign.md`

---

## 1. Problem Statement

Three gaps exist between the reference template (`template-po-v2.docx`) and the generated DOCX (`PO-112.docx`):

1. **Header renders as flat unstyled text in DOCX.** The current `buildBrowserDocumentHeaderHTML` uses `<div>`/`<span>` with flexbox. `HTMLtoDOCX` ignores flex layout and concatenates label+value text with no separator (e.g., `"ELABORADO PORleandro_theodoro"`).

2. **Body field sections render as flat paragraphs.** Field labels and values have no visual separation in DOCX output. The reference template uses bordered 2-column tables for every field section.

3. **Section 10 (Histórico de Revisões) is missing.** The reference has a 4-column version history table at the end. Our template body stops at Section 9.

---

## 2. Architectural Constraints

- Pipeline stays as-is: CKEditor HTML body → Go `generateBrowserDocxBytes` → `HTMLtoDOCX` → DOCX.
- `HTMLtoDOCX` (`@turbodocx/html-to-docx`) supports `<table>` with `background-color` on cells natively (maps to DOCX shading). `<div>` with flexbox is not supported.
- The React `DocumentEditorHeader.tsx` is not changed — it already renders correctly in the browser.
- All changes stay within: Go service layer, template seed (`template.go`), migrations, `document-content.css`.

---

## 3. Design

### 3.1 Header — Table Rewrite (Go)

**File:** `internal/modules/documents/application/service_content_docx.go`

`buildBrowserDocumentHeaderHTML` is rewritten from `<div>`/`<span>` to a single `<table>` with 3 rows:

| Row | Purpose | Background |
|-----|---------|------------|
| 1 — top bar | "Metal Nobre" (left) · "CODE · Rev. XX" (right) | `#6b1f2a` |
| 2 — title | Full document title (colspan 5) | `#3e1018` |
| 3 — meta | 5 cells: Tipo · Elaborado por · Data · Status · Aprovado por | `#3e1018` |

Each meta cell contains two `<p>` tags:
- `<p>` 1: label text (small, uppercase, white, opacity ~0.6)
- `<p>` 2: value text (normal weight, white)

All color, font-weight, padding, and border attributes are applied as inline styles on `<td>` and `<p>` elements. `HTMLtoDOCX` maps `background-color` on `<td>` to native DOCX shading, `color` to run color, `font-weight:bold` to bold run — these are all supported.

The full table structure:

```html
<table style="width:100%;border-collapse:collapse;font-family:DM Sans,sans-serif;">
  <tr>
    <td style="background-color:#6b1f2a;color:#fff;padding:6px 14px;font-size:11px;font-weight:600;letter-spacing:1px;text-transform:uppercase;">
      Metal Nobre
    </td>
    <td style="background-color:#6b1f2a;color:#fff;padding:6px 14px;font-size:11px;font-weight:600;text-align:right;white-space:nowrap;">
      CODE · Rev. XX
    </td>
  </tr>
  <tr>
    <td colspan="2" style="background-color:#3e1018;color:#fff;padding:10px 14px;font-size:16px;font-weight:700;">
      TITLE
    </td>
  </tr>
  <tr>
    <!-- 5 meta cells (colspan 2 total), each with label p + value p -->
    <!-- Implemented as a nested table or using a single row spanning both columns -->
    <td colspan="2" style="background-color:#3e1018;color:#fff;padding:0;">
      <table style="width:100%;border-collapse:collapse;">
        <tr>
          <td style="padding:6px 14px;border-right:1px solid rgba(255,255,255,0.18);width:20%;">
            <p style="margin:0;font-size:10px;font-weight:600;text-transform:uppercase;opacity:0.62;">Tipo</p>
            <p style="margin:0;font-size:12px;">VALUE</p>
          </td>
          <!-- Elaborado por, Data, Status, Aprovado por cells follow same pattern -->
        </tr>
      </table>
    </td>
  </tr>
</table>
```

All user-controlled field values are wrapped in `html.EscapeString`. The function signature and callers are unchanged.

**Tests updated:** `TestBuildBrowserDocumentHeaderHTML`, `TestBuildBrowserDocumentHeaderHTMLEmptyFields`, `TestBuildBrowserDocumentHeaderHTMLEscapesSpecialChars` — assertions updated to expect `<table>` structure instead of `<div>` structure.

---

### 3.2 Template Body — Field Layout (CSS + Template)

#### 3.2.1 New CSS Rule: `.md-field-table`

**File:** `frontend/apps/web/src/styles/document-content.css`

```css
/* ── Field Table ────────────────────────────────────────────── */
.ck-content .md-field-table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 1rem;
}

.ck-content .md-field-table td {
  border: 1px solid #dfc8c8;
  padding: 0.5rem 0.75rem;
  vertical-align: top;
  font-size: 0.93rem;
  color: #483030;
}

.ck-content .md-field-table .md-field-label {
  width: 30%;
  font-weight: 600;
  font-size: 0.84rem;
  color: #483030;
  background: #f9f3f3;
}

.ck-content .md-field-table td:last-child {
  width: 70%;
}

.ck-content .md-field-table .restricted-editing-exception {
  margin: 0;
  background: transparent;
  border: none;
  border-radius: 0;
  padding: 0;
  min-height: unset;
}
```

The old `.md-field` rules remain for backward compatibility with any existing saved content.

#### 3.2.2 Template Body Changes

**Files:** `internal/modules/documents/domain/template.go`, `migrations/0057_seed_po_browser_template.sql`, `migrations/0058_update_po_browser_template.sql`

Sections 2, 3, and 6 convert from `div.md-field` stacks to `table.md-field-table` rows:

**Before (Section 2 example):**
```html
<div class="md-field">
  <p class="md-field-label">Objetivo</p>
  <p class="restricted-editing-exception">Descreva o objetivo...</p>
</div>
```

**After:**
```html
<table class="md-field-table">
  <tbody>
    <tr>
      <td class="md-field-label">Objetivo</td>
      <td><p class="restricted-editing-exception">Descreva o objetivo deste procedimento, incluindo o resultado esperado ao final da execução.</p></td>
    </tr>
    <tr>
      <td class="md-field-label">Escopo</td>
      <td><p class="restricted-editing-exception">Defina os limites de aplicação deste procedimento: onde começa, onde termina e o que está fora do escopo.</p></td>
    </tr>
    <!-- all fields for the section in one table -->
  </tbody>
</table>
```

All fields within a section are grouped into a single table (not one table per field).

Sections unchanged: 4 (Visão Geral — `md-free-block`), 5 (Etapas — `md-free-block`), 7 (KPIs — `md-table`), 8 (Referências — `md-table`), 9 (Glossário — `md-table`).

---

### 3.3 Section 10 — Histórico de Revisões

Added at the end of the template body (Sections 2–9 remain in order):

```html
<div class="md-section">
  <h2>10. Histórico de Revisões</h2>
  <div class="md-table">
    <table>
      <thead>
        <tr>
          <th>Versão</th>
          <th>Data</th>
          <th>O que foi alterado</th>
          <th>Por</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td><p class="restricted-editing-exception">{{versao}}</p></td>
          <td><p class="restricted-editing-exception">{{data_criacao}}</p></td>
          <td><p class="restricted-editing-exception"></p></td>
          <td><p class="restricted-editing-exception">{{elaborador}}</p></td>
        </tr>
      </tbody>
    </table>
  </div>
</div>
```

- `{{versao}}`, `{{data_criacao}}`, `{{elaborador}}` are token placeholders substituted at bundle-serve time (see §3.4).
- "O que foi alterado" cell is an empty `restricted-editing-exception` — user types freely.
- All four cells are editable so the user can correct values if needed.
- Users duplicate rows in CKEditor to add more revision entries.

---

### 3.4 Bundle Token Substitution (Go — Service Layer)

**File:** `internal/modules/documents/application/service_browser_editor.go` (or equivalent bundle-serve function)

When building the browser editor bundle response, before setting `body`, the service performs a simple string substitution on the body content:

```go
body = substituteTemplateTokens(body, doc, version)
```

```go
func substituteTemplateTokens(body string, doc domain.Document, version domain.Version) string {
    versao := fmt.Sprintf("%02d", version.Number)
    data   := doc.CreatedAt.Format("02/01/2006")
    por    := doc.OwnerID
    if por == "" { por = "—" }

    body = strings.ReplaceAll(body, "{{versao}}", versao)
    body = strings.ReplaceAll(body, "{{data_criacao}}", data)
    body = strings.ReplaceAll(body, "{{elaborador}}", por)
    return body
}
```

- Substitution is idempotent: if tokens are already replaced (draft was saved after first load), the `ReplaceAll` calls are no-ops.
- Tokens are not HTML-escaped in the template (they render as text content of `<p>` tags, not as attributes).
- The substituted values are plain text — no HTML injection risk since document metadata is backend-owned, not user-supplied at this point.

**Tests added:** `TestSubstituteTemplateTokens` — verifies all three tokens are replaced; verifies idempotency when tokens are absent.

---

### 3.5 Migration Strategy

Since this project is pre-production with no live user data:

- `migrations/0057_seed_po_browser_template.sql` — updated in place with the new body HTML.
- `migrations/0058_update_po_browser_template.sql` — updated UPDATE statement to apply the new body to any existing rows.
- No new migration number needed.

Go/SQL parity test (`TestPOBrowserTemplateGoSQLParity`) continues to validate that `template.go` and `0057` are in sync.

---

## 4. What Does Not Change

- `generateBrowser.ts` (docgen service) — no changes.
- `DocumentEditorHeader.tsx` — no changes.
- `BrowserDocumentEditorView.tsx` — no changes.
- `DocumentEditorHeader.module.css` — no changes.
- Sections 4, 5, 7, 8, 9 of the template body — no changes.
- API contracts — no changes.
- The `restricted-editing-exception` mechanism — unchanged, just relocated into `<td>` cells.

---

## 5. Acceptance Criteria

1. `go test ./...` passes including `TestPOBrowserTemplateGoSQLParity` and `TestSubstituteTemplateTokens`.
2. Exported DOCX header renders as a 3-row colored table (not flat paragraphs).
3. Sections 2, 3, 6 in exported DOCX render as 2-column bordered tables.
4. Section 10 appears in the exported DOCX with Versão, Data, Por populated from document metadata.
5. Browser editor shows Section 10 with substituted values on first load.
6. "O que foi alterado" cell is blank and editable.
7. All `restricted-editing-exception` zones remain editable in CKEditor with no regressions.
8. TypeScript build passes (`npx tsc --noEmit`).
