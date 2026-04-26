# ADR 0008 — Replace user-fill placeholders with fixed catalog

> **Last verified:** 2026-04-26
> **Status:** Accepted
> **Date:** 2026-04-26
> **Scope:** Replacing user-fill placeholder types with a fixed catalog of 7 computed tokens.

## Context

The previous placeholder model (post-migration 2026-04-25) supported multiple `type` values: `text`, `date`, `number`, `select`, `user`, `picture`, `computed`. Authors could define arbitrary placeholder names and types; end users filled them in via a form panel in the document editor.

Problems with that model:

1. **Complexity without payoff.** The fill-in form required type-specific UI (date pickers, select dropdowns, length constraints) that added frontend and backend surface area for a feature still in early use.
2. **Eigenpal autosave constraint.** Eigenpal autosaves the DOCX on every change (1500ms debounce). If `applyVariables` were called in the editor to show a substituted preview, the next autosave would persist the substituted DOCX — destroying the original `{token}` strings. There is no two-buffer model today (raw edit buffer + substituted preview buffer), so in-editor substitution is unsafe.
3. **Metadata drift.** User-fill types introduced a schema layer (label, type, constraints, options) that could drift from the DOCX tokens, creating orphan metadata.
4. **All real use cases are computed.** Every actual token MetalDocs needs (doc code, title, revision, author, dates, approvers, area) is derived from existing structured data — none requires freeform user input.

## Decision

**Replace the open-ended placeholder model with a fixed catalog of 7 computed tokens.**

Catalog:

| Token | Resolver source |
|---|---|
| `{doc_code}` | Document code from profile sequence counter |
| `{doc_title}` | Document title field |
| `{revision_number}` | Revision counter on the document version |
| `{author}` | Display name of document author |
| `{effective_date}` | Effective date set at approval/freeze |
| `{approvers}` | Approver names joined by `", "`; `"[aguardando aprovação]"` if empty |
| `{controlled_by_area}` | Area name from the document's taxonomy binding |

Rules:
- All catalog tokens are `computed` — no user input.
- Template authors type `{token}` in the DOCX; the catalog panel auto-detects (via `getVariables()`) and saves them as computed entries.
- The backend rejects non-catalog names at schema-save via `ValidatePlaceholders`.
- There is no fill-in panel in the document editor.
- Tokens remain literal in the DOCX until finalize/freeze.
- Substitution occurs server-side at freeze via the existing fanout pipeline.

### `applyVariables` — explicitly deferred

`applyVariables` (eigenpal browser-side substitution API) is NOT called in writer mode. Rationale: autosave would persist the substituted DOCX, destroying original tokens. This API is reserved for a future "preview mode" with a two-buffer story:
- **Edit buffer:** always holds raw `{token}` DOCX (autosaved to server).
- **Preview buffer:** ephemeral in-memory substituted copy (never autosaved).

That two-buffer design is out of scope for this ADR and should be addressed in a future decision when preview mode is prioritised.

## Consequences

**Positive:**
- Simpler model: no fill-in types, no constraint schema, no fill-in form panel.
- No metadata drift: catalog is static; the schema just records which catalog tokens are present in the current DOCX.
- Backend validation is straightforward: compare detected names against a constant set.
- No autosave risk: tokens stay literal in the editor DOCX.

**Negative / trade-offs:**
- Freeform user-fill placeholders are no longer supported. If a future use case genuinely requires user input (e.g., a "remarks" field), a new token type or a separate mechanism must be designed.
- `ApproversResolver` returns a flat string (`names joined by ", "`). Rich formatting (e.g., per-approver signature blocks) is not possible without a more complex resolver or a template section tag (`{#approvers}`…`{/approvers}`).

## Verification

- Branch: `feat/placeholder-fixed-catalog`
- Backend: `ValidatePlaceholders` unit-tested; `ApproversResolver` returns `"[aguardando aprovação]"` for empty list.
- Frontend: catalog panel auto-detects tokens; non-catalog names surface a validation error before save.

## Cross-refs

- [concepts/placeholders.md](../concepts/placeholders.md) — updated catalog doc
- [modules/editor-ui-eigenpal.md](../modules/editor-ui-eigenpal.md) — `applyVariables` not used in writer mode
- [decisions/0003-token-syntax-migration.md](0003-token-syntax-migration.md) — prior migration to `{name}` syntax
- [decisions/0002-zone-purge.md](0002-zone-purge.md) — prior simplification (zones removed)
- Plan: `docs/superpowers/plans/2026-04-26-placeholder-fixed-catalog.md`
