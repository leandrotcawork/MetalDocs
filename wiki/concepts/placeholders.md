# Placeholders — Fixed Catalog Model

> _Changelog: 2026-04-26 — rewritten for fixed-catalog model (ADR 0008); dropped legacy fill-in workflow content._
>
> **Last verified:** 2026-04-26
> **Scope:** What a placeholder is, the fixed 7-entry catalog, how tokens stay literal in the editor, and when substitution occurs.
> **Out of scope:** Substitution engine internals (see `modules/render-fanout.md`), editor plugin wiring (see `modules/editor-ui-eigenpal.md`).
> **Key files:**
> - `packages/editor-ui/src/MetalDocsEditor.tsx:54` — eigenpal `templatePlugin` wired here
> - `frontend/apps/web/src/features/templates/v2/TemplateAuthorPage.tsx` — catalog panel, auto-detect via `getVariables()`
> - `frontend/apps/web/src/features/templates/placeholder-types.ts` — `CatalogPlaceholder` type
> - `internal/modules/templates_v2/application/validate_placeholders.go` — `ValidatePlaceholders` rejects non-catalog names
> - `internal/modules/render/fanout/` — server-side substitution at freeze/finalize (Go)
> - `internal/modules/render/fanout/resolvers/approvers_resolver.go` — `ApproversResolver`

---

## What a placeholder is

A `{token}` in a template DOCX that gets substituted with a computed value when a document is finalized (frozen). Example: `{doc_code}` → `"QMS-001-v2"`.

Tokens use single-brace `{name}` syntax — docxtemplater standard, detected natively by eigenpal's `templatePlugin`.

## The fixed catalog

MetalDocs defines exactly 7 computed tokens. Template authors may only use names from this list. The backend rejects any other name at schema-save time (`ValidatePlaceholders`).

| Token | Resolver source |
|---|---|
| `{doc_code}` | Document code — generated from profile sequence counter |
| `{doc_title}` | Document title field |
| `{revision_number}` | Revision counter on the document version |
| `{author}` | Display name of the document author (creator) |
| `{effective_date}` | Effective date set during approval/freeze |
| `{approvers}` | Approver names joined by `", "`; `"[aguardando aprovação]"` if none |
| `{controlled_by_area}` | Area name from the document's taxonomy binding |

All tokens are **computed** — no user input is required. There is no fill-in panel in the document editor.

## Authoring workflow

1. Template author types `{token}` directly in the DOCX inside the editor (or in Word desktop).
2. Eigenpal's `templatePlugin` auto-detects the token, highlights it orange, and lists it as a chip in the sidebar.
3. `TemplateAuthorPage` reads `editorRef.current.getAgent().getVariables()` after each editor change and auto-saves detected names as `computed` entries in the template schema.
4. The catalog panel shows what each detected token resolves to.
5. Non-catalog names are rejected by `ValidatePlaceholders` when the schema is saved.

## Tokens are literal until freeze

Tokens are **never substituted** in the editor (writer mode). The DOCX stored on disk always contains the raw `{token}` strings until a document is finalized.

Reason: eigenpal autosaves on every change. Calling `applyVariables` in-editor would mutate the DOCX with substituted values, destroying the original tokens on the next autosave cycle. See ADR 0008 for the full rationale and the deferred "preview mode" story.

Substitution happens exclusively at **finalize/freeze** via the existing server fanout pipeline:
1. `freeze_service.go` resolves each catalog token via its resolver.
2. The `{name: value}` map is passed to docxtemplater → native substitution.
3. The frozen DOCX (with resolved values) is archived and rendered to PDF.

## `applyVariables` — deferred

The eigenpal `applyVariables` API (browser-side substitution) is intentionally not called in writer mode. It is reserved for a future "preview mode" with a two-buffer story (edit buffer keeps raw tokens; preview buffer holds a substituted copy). See ADR 0008.

## Feature status

| Feature | Status |
|---|---|
| Token format | `{name}` — eigenpal native |
| Editor highlighting | orange via `templatePlugin` |
| Sidebar chips | eigenpal native |
| Catalog enforcement | backend `ValidatePlaceholders` rejects non-catalog names |
| Server substitution | docxtemplater at freeze/finalize |
| Fill-in panel | not present — all tokens are computed |
| `applyVariables` in editor | deferred (see ADR 0008) |

## History

- **Pre-2026-04-25:** MetalDocs used `{{uuid}}` double-brace tokens and had user-fill types (text/date/number/select). See `decisions/0003-token-syntax-migration.md`.
- **2026-04-25:** Migrated to `{name}` single-brace tokens; eigenpal authoring convergence.
- **2026-04-26:** Replaced user-fill placeholder model with fixed 7-entry computed catalog. See `decisions/0008-placeholder-fixed-catalog.md`.

## Cross-refs

- [concepts/token-syntax.md](token-syntax.md) — deeper dive on `{name}` vs `{{uuid}}`
- [modules/editor-ui-eigenpal.md](../modules/editor-ui-eigenpal.md) — how MetalDocsEditor wires eigenpal plugins
- [modules/render-fanout.md](../modules/render-fanout.md) — server substitution code
- [decisions/0008-placeholder-fixed-catalog.md](../decisions/0008-placeholder-fixed-catalog.md) — fixed catalog ADR
- [decisions/0003-token-syntax-migration.md](../decisions/0003-token-syntax-migration.md) — token syntax migration ADR
