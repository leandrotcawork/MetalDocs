# Eigenpal Spike

> **Last verified:** 2026-04-25 (authoring convergence)
> **Scope:** What the eigenpal spike was, where it lives, what each test (T1–T8) verified, what carried over to MetalDocs.
> **Out of scope:** Current MetalDocs eigenpal integration (see `modules/editor-ui-eigenpal.md`).
> **Key files:**
> - `C:\Users\leandro.theodoro.MN-NTB-LEANDROT\Documents\eigenpal-spike\RESULTS.md` — top-level spike conclusions
> - `C:\Users\leandro.theodoro.MN-NTB-LEANDROT\Documents\eigenpal-spike\spike\src\pages\T*.tsx` — per-test page components
> - `C:\Users\leandro.theodoro.MN-NTB-LEANDROT\Documents\eigenpal-spike\public\fixtures\` — DOCX fixtures (e.g., `placeholders.docx`)

---

## Why the spike existed

Pre-decision: should we adopt `@eigenpal/docx-js-editor` to replace CKEditor / BlockNote? Spike validated each capability we needed before committing.

## Tests

| # | Capability | Result | Carried into MetalDocs? |
|---|-----------|--------|--------------------------|
| T1 | Restricted editing zones (Word `<w:permStart/End>`) | ❌ FAIL — eigenpal ignores Word's permission XML | Skipped. Zones became MetalDocs custom (now purged). |
| T2 | Round-trip DOCX integrity | ✅ PASS | Trusted — used as base assumption |
| T3 | Comments & track changes | ✅ PASS (with caveats) | Wired to MetalDocs comments API |
| T4 | **templatePlugin** — placeholder detect + substitute | ✅ PASS | **Foundation:** eigenpal native `{name}` syntax, `getAgent().getVariables()`, and `applyVariables()`. Used by MetalDocs authoring sync. |
| T5 | Schema fill panel (form per type) | ✅ PASS | MetalDocs reimplements server-side |
| T6 | Metadata via eigenpal plugin | ⚠️ Partial | MetalDocs uses own toolbar instead |
| T7 | **OutlinePlugin** — heading nav | ✅ PASS (with perf caveat at 1.7MB+) | **Ported and fixed** — module-level singleton bug → factory + `useMemo`. Lives at `packages/editor-ui/src/plugins/OutlinePlugin.tsx`. |
| T8 | Browser-side publish pipeline | ✅ PASS | Skipped — Gotenberg server-side instead |

## Key fixture

`spike/public/fixtures/placeholders.docx` — minimal DOCX with eigenpal-native tokens:
```
{doc_title}
{author_name}
{rev_number}
{effective_date}
```

Single-brace, semantic names. **This is the truth** for what eigenpal expects. T4 verified end-to-end:
1. Load fixture
2. Eigenpal auto-detects all 4 tokens, highlights orange
3. `editor.getAgent().getVariables()` returns the 4 names
4. `editor.getAgent().applyVariables({...}).then(a => a.toBuffer())` produces substituted DOCX
5. Downloaded DOCX opens in Word with values substituted ✅

## What MetalDocs took

**Adopted directly:**
- `@eigenpal/docx-js-editor` as the only editor (no CKEditor / BlockNote)
- ProseMirror underpinning
- Eigenpal's autosave callback pattern → wrapped with debounce in `MetalDocsEditor`
- T7 outline plugin (with the singleton-bug fix)

**Reimplemented in MetalDocs:**
- Placeholder authoring UI (custom Variables panel) — Converged 2026-04-25 — token in DOCX is source of truth, schema auto-syncs
- Server-side substitution via Go fanout (instead of T4's browser `applyVariables()`) — keeps audit canonical
- Metadata fields (toolbar instead of T6's plugin)

**Tried and abandoned (pre-MetalDocs):**
- T1 restricted zones → became MetalDocs "editable zones" (custom) → purged 2026-04-25 (`decisions/0002-zone-purge.md`)

## Drift watch

The spike repo is FROZEN reference. If we update eigenpal version in MetalDocs and the API changes, the spike won't catch it — re-validate against current `@eigenpal/docx-js-editor` types in `frontend/apps/web/node_modules/@eigenpal/docx-js-editor/dist/*.d.ts`.

## Cross-refs

- [concepts/placeholders.md](../concepts/placeholders.md) — T4 finding deep-dive
- [modules/editor-ui-eigenpal.md](../modules/editor-ui-eigenpal.md) — current integration
- [decisions/0001-eigenpal-adoption.md](../decisions/0001-eigenpal-adoption.md) — adoption ADR
- [decisions/0002-zone-purge.md](../decisions/0002-zone-purge.md) — T1-related zone removal
