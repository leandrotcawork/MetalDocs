# Placeholders — Eigenpal Native + MetalDocs Metadata

> **Last verified:** 2026-04-25 (authoring convergence)
> **Scope:** What a placeholder is, how eigenpal handles it natively, how MetalDocs uses it as the source of truth for authoring.
> **Out of scope:** Fill-in form UI (see `workflows/document-fillin.md`), substitution at render time (see `modules/render-fanout.md`).
> **Key files:**
> - `packages/editor-ui/src/MetalDocsEditor.tsx:54` — eigenpal `templatePlugin` wired here
> - `frontend/apps/web/src/features/templates/v2/TemplateAuthorPage.tsx` — `getAgent().getVariables()` schema sync and `insertTokenAtCursor(\`{${placeholder.name}}\`)`
> - `frontend/apps/web/src/features/templates/placeholder-types.ts` — Placeholder schema type (id, name, label, type, constraints)
> - `frontend/apps/web/src/features/templates/placeholder-chip.tsx` — drag-drop chip UI
> - `internal/modules/render/fanout/` — server-side substitution (Go)
> - `C:\Users\leandro.theodoro.MN-NTB-LEANDROT\Documents\eigenpal-spike\spike\src\pages\T4TemplatePlugin.tsx` — spike T4 reference impl
> - `C:\Users\leandro.theodoro.MN-NTB-LEANDROT\Documents\eigenpal-spike\public\fixtures\placeholders.docx` — fixture using `{name}` syntax

---

## What a placeholder is

A variable in a template DOCX that gets substituted with a real value when an end user fills in a document. Example: `{customer_name}` → `"Acme Corp"`.

## Eigenpal-native mechanism (the truth)

**Token syntax:** Single brace `{name}` — docxtemplater standard.

**Detection:** Eigenpal's `templatePlugin` (a ProseMirror plugin) auto-scans the document on every change. Tokens matching the docxtemplater regex are:
- Highlighted orange in the canvas
- Listed as chips in the eigenpal sidebar
- Tracked as `TemplateTag` objects with `{ id, type, name, rawTag, from, to }`

Tag types eigenpal supports:
- `variable` — `{name}` simple substitution
- `sectionStart` / `sectionEnd` — `{#name}...{/name}` repeating sections
- `invertedStart` — `{^name}...{/name}` conditional
- `raw` — `{@name}` raw HTML/OOXML injection

**Substitution API (browser-side):**
```ts
const agent = editorRef.current.getAgent();
const variables = agent.getVariables();              // discover
const filled = await agent.applyVariables({          // substitute
  customer_name: 'Acme Corp',
  effective_date: '2026-04-25',
});
const buffer = await filled.toBuffer();              // export DOCX
```

This was verified end-to-end in eigenpal-spike T4 with fixture `placeholders.docx`. See `references/eigenpal-spike.md`.

**Backend substitution:** Same docxtemplater algorithm. Identical syntax. So a template authored with `{name}` tokens works for both browser preview AND server fanout — single source of truth.

## MetalDocs current mechanism (converged, 2026-04-25)

**Token syntax:** Single brace `{name}` — docxtemplater standard, matching eigenpal native.

**Authoring source of truth:** the DOCX token exists if and only if the author typed `{name}` in the document. `TemplateAuthorPage.tsx` calls `editorRef.current.getAgent().getVariables()` after editor changes and auto-creates missing schema metadata entries for detected token names.

**Insertion:** Direct typing is preferred. Clicking or dragging an existing placeholder chip still inserts `` `{${placeholder.name}}` `` via `insertTokenAtCursor()`.

**Detection:** Eigenpal's `templatePlugin` detects `{name}` tokens and:
- Highlights them orange in the canvas
- Shows them as chips in the eigenpal sidebar

**Orphans:** If schema metadata exists for a placeholder name that is no longer present in `getVariables()`, MetalDocs keeps the schema entry but marks the chip as orphan. Authors can remove that stale metadata from the inspector.

**Schema metadata** (`placeholder-types.ts`):
```ts
type Placeholder = {
  id: string;           // UUID — internal PK only, never appears in DOCX
  name: string;         // slug — used as the DOCX token (e.g. "customer_name")
  label: string;        // human-readable name shown in UI
  type: 'text' | 'date' | 'number' | 'select' | 'user' | 'picture' | 'computed';
  required?: boolean;
  maxLength?: number;
  options?: string[];   // for type=select
  resolverKey?: string; // for type=computed
  // ...constraints depending on type
}
```

`name` is validated: unique per template version, matches `^[a-z][a-z0-9_]{0,49}$`. Auto-derived from `label` by `slugifyLabel()` in the inspector.

**Substitution:** Server-only. The flow:
1. User fills placeholder values via custom form
2. `freeze_service.go` builds a `{name: value}` map (keyed by slug, not UUID)
3. Server fanout passes map to docxtemplater — native `{name}` substitution

## Feature status

| Feature | Status |
|---------|--------|
| Token format | ✅ `{name}` — eigenpal native |
| Editor highlighting | ✅ orange via `templatePlugin` |
| Sidebar chips | ✅ eigenpal native |
| Server substitution | ✅ docxtemplater (name-keyed map) |
| Constraints (required, regex, min/max) | ✅ MetalDocs schema layer |
| Authoring convergence | ✅ DOCX token is source of truth; schema auto-syncs metadata |
| Word desktop authoring | ✅ type `{customer_name}` directly |

## History

MetalDocs originally used `{{uuid}}` (double-brace UUID) tokens. `templatePlugin` was wired but inert because it only detects `{name}`. Migrated 2026-04-25. See `decisions/0003-token-syntax-migration.md` for the full ADR and rationale.

## Cross-refs

- [token-syntax.md](token-syntax.md) — deeper dive on `{name}` vs `{{uuid}}`
- [modules/editor-ui-eigenpal.md](../modules/editor-ui-eigenpal.md) — how MetalDocsEditor wires eigenpal plugins
- [modules/render-fanout.md](../modules/render-fanout.md) — server substitution code
- [decisions/0003-token-syntax-migration.md](../decisions/0003-token-syntax-migration.md) — migration ADR
- [references/eigenpal-spike.md](../references/eigenpal-spike.md) — T4 spike result
