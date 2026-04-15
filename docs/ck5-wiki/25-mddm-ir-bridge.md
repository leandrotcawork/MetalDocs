---
title: MDDM IR bridge — HTML as source of truth
status: draft
area: storage
priority: HIGH
---

# 25 — CK5 HTML ↔ MDDM structured representation

CKEditor 5's only stable public serialization is an **HTML string**
(`editor.getData()` / `editor.setData()`; see
[24 — Data format](./24-data-format.md)). The model tree is internal, and the
view layer is split between an editing view (with widget chrome) and a data
view (clean HTML). MetalDocs currently ships a structured **MDDM IR** under
`frontend/apps/web/src/features/documents/mddm-editor/engine/` with a
docx-emitter, pdf-exporter, asset-resolver, ir-hash, and golden tests. That
IR was designed around BlockNote, before we pinned CK5 v48.

This page recommends a **CK5-native** architecture for how document content,
templates, fill-mode state, and exports relate. Prior MDDM conventions
(IR-as-SoT, ir-hash completeness gate) are treated as **reference**, not
binding — see the [Deviations](#deviations-from-prior-mddm-conventions)
section for where we break with them.

---

## Recommended architecture — TL;DR

1. **Source of truth: Option C — HTML canonical, IR ephemeral.**
   The backend stores exactly one artefact per document/template: the CK5
   HTML produced by `editor.getData()`. An intermediate structured tree is
   computed **on demand** during export (DOCX/PDF), and thrown away after.
   There is no parallel content store, no ir-hash gate, no IR→HTML emitter.

2. **Parser: direct DOM-lib parse (linkedom / parse5), not headless CK5.**
   Export runs in Node, reads the HTML string, and walks it with a
   lightweight DOM parser. Schema conformance is enforced at
   **ingress** (when HTML enters CK5 on load; see §10 Validation) rather
   than at export time.

3. **Emitter: none. Templates are authored as HTML from day one.**
   The "IR → HTML" hydration step goes away. The template IS its CK5 HTML.
   Fill mode loads that HTML via `setData()`; restricted-editing markers
   round-trip via `markerToData`.

4. **DOCX: keep the existing `docx-emitter` tree-walker, re-pointed at the
   parsed HTML** (not at layout-IR). Existing goldens remain an asset;
   bespoke IR nodes that no longer have an HTML equivalent retire cleanly.

5. **PDF: browser-side print (Paged.js / CSS Paged Media) on the same HTML,
   driven by the existing print-stylesheet.** Headless Chromium remains an
   option for server-side PDF, but is no longer required for parity with
   the editor's visual output.

6. **Field values: stored as marker text content inside the HTML**, with
   an optional sidecar `{ fieldId: value }` map for fast server-side
   querying only. The HTML remains self-contained and authoritative.

7. **Versioning: schema version attribute on the root wrapper +
   upcast-time migrations**, replacing ir-hash drift detection.

The rest of this page walks each decision.

---

## 1. Source of truth

### The three options (recap)

- **A — CK5 HTML is SoT; exporters consume HTML directly.**
- **B — MDDM IR is SoT; CK5 HTML is transport; bidirectional bridge.**
- **C — Hybrid: HTML canonical persist; IR is an ephemeral projection.**

### Recommendation: **C**

**Why not B (status quo).** Keeping IR as SoT means every author keystroke
in CK5 must produce a valid IR through a bridge. That bridge has to cover
**every** CK5 feature we ship — GHS fallbacks, nested tables, markers,
restricted-editing exceptions, image widgets, list markers, paste-from-Word
artefacts — and keep the two trees in lockstep. CK5's data pipeline already
does this job, and does it well, against its own schema. Duplicating it
against a second tree is the exact kind of "parallel content model" that
burns teams maintaining rich editors. The ir-hash gate is the symptom, not
the fix.

**Why not A (pure HTML, zero intermediate).** We still need a structured
walk for DOCX — `docx` (the npm library) is node-based, not HTML-string
based, and the existing goldens depend on a stable visit order. An
ephemeral tree computed at export time gives us that structure without
making it a persistence contract.

**Why C.** HTML is what CK5 round-trips losslessly for the content covered
by our schema (see
[24 — Data format §6](./24-data-format.md)). It is the only representation
the editor guarantees. Everything else — IR, field-value map, preview
cache — is derivable from it. If derivable artefacts drift, we regenerate.

### Practical shape

- Backend table: one `content_html TEXT` column per document + per template.
- Optional `content_meta JSONB` for derived data that is expensive to
  compute (extracted field values, outline, word count). Always
  regeneratable from `content_html`.
- No `content_ir` column. No ir-hash. No completeness-gate check on write.

### Deviations from prior MDDM conventions

The MDDM sprint memory locked IR-as-SoT with a completeness gate. This page
explicitly reverses that call. Rationale:

- CK5's upcast pipeline is the strongest possible completeness gate — it is
  the editor's own view of "what survives". A second gate on top of it is
  redundant and fragile.
- Templates are CK5-authored; there is no external producer that needs an
  IR input format.
- DOCX/PDF exporters read the same HTML the editor reads. One surface, one
  parse path.

---

## 2. Parser strategy (HTML → structured, server-side)

Two candidates:

### (a) Headless CK5 in Node

Load the same `ckeditor5` plugin set server-side, `setData(html)`, then walk
`editor.model.document.getRoot()`. Gives us the exact schema guarantee: if
the model contains it, it is valid.

Costs:

- **Bundle and startup**: CK5 is ~400–600 KB gzipped on the browser; the
  Node evaluation needs a DOM shim (jsdom / linkedom). See
  [38 — Build & bundling §8](./38-build-bundling.md) — CK5 touches `window`
  at import time, so Node use requires a global shim.
- **Cold start** dominates export latency for on-demand PDF/DOCX (tens to
  low hundreds of ms per worker process).
- **SSR caveats** from page 02: "CK5 is a browser-only library". Running
  it under Node is supported in principle but not the documented path.

### (b) Direct DOM-lib parse (linkedom / parse5 / cheerio)

Parse the HTML string with a spec-compliant HTML parser, walk the resulting
DOM, emit our export tree.

Costs:

- No schema guarantee at parse time — if invalid HTML slipped through, the
  walker sees it.
- We own the element-name → export-node mapping instead of reading it from
  CK5's schema.

### Recommendation: **(b), with ingress-time schema enforcement.**

We enforce schema once at ingress (on `setData` in the editor + optional
server-side canonicalization, §10). After that, the stored HTML is
**already** schema-valid by construction. The export-time parser can be
dumb and fast; it does not need CK5's schema on standby.

Library choice: **linkedom** (fast, small, DOM-API compatible) or
**parse5** (spec-strict, lower-level). Prefer linkedom for ergonomics;
fall back to parse5 if we hit HTML5 edge cases linkedom mishandles.
⚠ uncertain — benchmark both on a representative 50-page template before
committing.

---

## 3. Emitter strategy (structured → HTML, for hydration)

If templates are authored in CK5, **the template is HTML from day one**.
There is no "upstream IR" that needs to be rendered into CK5's format.

Consequences:

- No `IR → HTML` emitter. Drop `engine/layout-interpreter/` once migration
  completes.
- No "render template into editor" step beyond `editor.setData(templateHtml)`.
- Fill mode hydration is a single `setData()` call plus
  `editor.execute('goToNextRestrictedEditingException')` (see §7).

**Pushback on the task prompt's option:** the prompt asks whether we need
an IR → HTML emitter at all. Answer: no. Only keep one if we introduce a
non-CK5 producer (e.g. a CLI that generates templates from JSON). That is
not on the roadmap.

---

## 4. What the intermediate tree is (if we keep one)

We keep exactly **one** intermediate tree, and only for export:

```
HTML  ──parse──►  ExportNode tree  ──visit──►  DOCX / PDF
```

Properties of `ExportNode`:

- **Narrow**: covers only nodes relevant to paginated output
  (sections, paragraphs, headings, lists, tables, images, breaks, fields).
- **Flat attributes**: no deep nesting of presentational metadata; styling
  is derived from the print-stylesheet, not carried in the tree.
- **Stateless**: computed fresh on every export. No store, no cache
  invalidation, no hash.
- **Export-only**: never shown to the user, never edited, never persisted.

This is **not** the old MDDM IR. The old IR tried to be a portable
interchange format; this tree is a disposable visitor input.

⚠ uncertain — whether we can reuse any of the existing `layout-ir` types as
`ExportNode`s directly, or whether a fresh set is cleaner. Likely a
partial reuse: block/text/table shapes survive; template/fill-specific
nodes retire.

---

## 5. DOCX export path

### Options

1. **Keep `docx-emitter/` over a structured tree.** Feed it the parsed
   HTML's `ExportNode` tree instead of the layout-IR. Most of the visitor
   code survives, only the input adapter changes.
2. **Rewrite against parsed HTML directly.** A DOM-walker → `docx`
   (npm) builder. Loses the current visitor abstraction; roughly as much
   code, less structure.
3. **Use `html-docx-js`.** Trivial, produces a legal `.docx`, but gives us
   no control over styling and cannot honour our print-stylesheet or
   pagination rules.
4. **`mammoth` (reverse direction).** N/A — mammoth is DOCX → HTML.

### Recommendation: **Option 1.**

- Preserves the docx-emitter visitor pattern and its golden tests.
- Existing goldens are an asset; any plan must satisfy them or
  intentionally retire specific cases. Option 1 lets us retire on a
  case-by-case basis with visible diffs.
- Input becomes "parsed HTML → ExportNode" (see §2, §4); output pipeline
  unchanged.
- No new dependencies; `docx` (npm) stays.

Migration cost: rewrite only the input adapter. Existing node emitters
(paragraph, heading, table-cell, run, image) keep their signatures.

⚠ uncertain — how many existing goldens depend on IR-only concepts
(pagination hints, completeness metadata) that the HTML round-trip no
longer carries. Audit before cutover; retire with PR notes.

---

## 6. PDF export path

### Current state

Layout-IR → print-stylesheet → PDF (via headless Chromium, assumed).

### Options

1. **Browser-side print (CSS Paged Media / Paged.js).** Render the CK5
   data view HTML inside a print container in the browser, apply
   `@page` + the existing print-stylesheet, trigger `window.print()` or
   save via a save-as-PDF command.
2. **Headless Chromium on the server.** Same HTML, same CSS, rendered via
   Puppeteer/Playwright; produces a byte-identical PDF to the browser
   path.
3. **Direct PDF library** (`pdfmake`, `pdfkit`). Gives up CSS parity.
4. **`html-pdf-chrome` / `@react-pdf/renderer`.** Partial coverage, not
   worth the porting cost.

### Recommendation: **Option 2 as default; Option 1 for in-browser
"Download PDF".**

- Option 2 gives us one rendering engine (Chromium) for both preview and
  export; pagination behaves identically to what the user sees.
- Chromium is Apache-2 licensed; no GPL conflict with our CK5 GPL use
  (see [01 — license](./01-license-gpl.md)) — the editor and the PDF
  renderer are separate processes.
- Option 1 becomes a future enhancement for offline/client-only flows.

⚠ uncertain — server operational cost of keeping a warm Chromium pool.
Benchmark against current pdf-exporter performance before committing.

---

## 7. Template hydration into fill mode

No IR expansion step. The sequence is:

1. Backend returns template HTML (CK5 data view, contains
   `restrictedEditingException` markers persisted as
   `data-restricted-editing-exception-*` attributes — see
   [07 — Markers §5](./07-markers.md)).
2. Frontend instantiates a fresh editor with `RestrictedEditingMode`
   (not `StandardEditingMode` — see
   [10 — Restricted editing](./10-restricted-editing.md); the two are
   mutually exclusive).
3. `editor.setData(templateHtml)` — CK5 upcasts the HTML, restoring both
   the model tree and the exception markers.
4. `editor.execute('goToNextRestrictedEditingException')` — caret lands
   in the first fillable region.

No IR reconstitution. No field expansion. No per-field placeholder
rendering on our side — the exception markers ARE the fields.

---

## 8. Field values and per-field data

Two candidate storage shapes:

### (a) Values inside the HTML

Marker text content carries the filled value. A filled template's HTML
contains, e.g.:

```html
<p>
  Customer:
  <span class="restricted-editing-exception">Acme Corp</span>
</p>
```

### (b) Lifted sidecar map

HTML carries placeholders; a separate `{ fieldId: "Acme Corp" }` map is
stored alongside.

### Recommendation: **(a) canonical; (b) derived.**

- The HTML stays self-contained — open a saved document and it renders
  without a second resource.
- DOCX/PDF export reads one artefact, not two. No "value not yet joined"
  race.
- For fast server-side query ("find all documents where `customerName =
  Acme"), compute a derived `{ fieldId: value }` map on save and index
  that. It is a cache, not a source of truth.
- Partial updates (user edits one field): CK5 fires `change:data`, we
  autosave the full HTML. Field granularity is not worth the complexity
  at document sizes we target (single-digit MB HTML).

⚠ uncertain — whether plain `<span class="restricted-editing-exception">`
is enough to identify a field, or whether we need an additional
`data-field-id` attribute for stable identity across edits. Leaning
toward adding `data-field-id` so that renames of the surrounding text do
not collide; resolve in the template-authoring page.

---

## 9. Versioning & migration

### Replace ir-hash with a schema-version attribute

Put a `data-mddm-schema="3"` attribute on the outermost wrapper element of
every stored document. On load:

```ts
// pseudo-code, server-side or in a pre-upcast step
const version = extractSchemaVersion(html);
const migrated = runMigrations(html, version, CURRENT);
editor.setData(migrated);
```

Migrations are pure HTML-to-HTML transforms (cheerio/linkedom), versioned
in a single `migrations/` folder, idempotent when re-run at their own
version.

### Preserving unknown attributes

Use **General HTML Support (GHS)** at low priority to preserve attributes
we do not yet understand, so that a document written by a newer client
does not lose data when opened in an older client. See
[24 — Data format §5](./24-data-format.md): anything without a converter
is dropped unless GHS keeps it.

Caveat: v47.6.0 had a GHS XSS advisory
(see [02 — Versioning](./02-versioning.md)). Pin to v48+ and constrain
GHS to an allow-list of attribute prefixes (`data-mddm-*`,
`data-restricted-*`).

---

## 10. Validation and drift

### Replace ir-hash with ingress-time schema conformance

- **Client-side**: the editor's own schema filter is the first line of
  defence. Invalid HTML becomes valid-by-dropping on `setData`.
- **Server-side canonicalization (optional, recommended)**: on write,
  run the HTML through a headless CK5 once (§2 option a), do
  `setData → getData`, and store the result. This produces a stable,
  canonicalized form and detects drift as a side effect (input ≠ output
  means something was dropped — log and surface).
- **Export-time**: no validation. Trust the stored HTML.

This replaces the completeness-gate role of `ir-hash` with a single
canonicalization step at the boundary where drift can actually happen
(save), rather than everywhere.

⚠ uncertain — canonicalization cost per write. If it's over a few hundred
ms for a typical document, make it async (accept write, canonicalize in
background, compare, alert on drift).

---

## Deviations from prior MDDM conventions

| Prior convention                          | This page                                               | Why                                                        |
|-------------------------------------------|---------------------------------------------------------|------------------------------------------------------------|
| MDDM IR is source of truth                | CK5 HTML is source of truth                             | HTML is CK5's only stable public format; IR duplicates it. |
| ir-hash completeness gate                 | Ingress canonicalization + schema version               | Gate belongs at the boundary, not at every read.           |
| IR → HTML emitter for editor hydration    | Templates are HTML from day one; no emitter             | CK5 is the template authoring tool; no upstream producer.  |
| Export consumes IR                        | Export consumes parsed HTML via disposable ExportNode   | One persistence surface; export tree is derivable.         |
| Field values tracked in parallel JSON     | Field values inside HTML (marker content); JSON is cache| Self-contained HTML; fewer join races on export.           |

---

## Open questions

- ⚠ uncertain — linkedom vs parse5 performance/fidelity on a full 50-page
  template. Needs a benchmark harness reusing an existing golden.
- ⚠ uncertain — how many DOCX goldens survive the IR-to-HTML re-pointing
  without intentional changes. Audit needed before cutover.
- ⚠ uncertain — whether we should add `data-field-id` to the
  restricted-editing exception span (§8) or rely on marker group naming
  alone.
- ⚠ uncertain — canonicalization cost at write time (§10) and whether to
  run it sync or async.
- ⚠ uncertain — tag emitted by block restricted-editing exception
  (almost certainly `<div>`, per
  [10 — Restricted editing](./10-restricted-editing.md)). Empirical check
  on pinned v48 pending.

---

## Sources & cross-refs

- [01 — License (GPL)](./01-license-gpl.md)
- [02 — Versioning & v48 distribution](./02-versioning.md)
- [03 — Engine model](./03-engine-model.md)
- [04 — Schema](./04-schema.md)
- [05 — Conversion](./05-conversion.md)
- [07 — Markers](./07-markers.md)
- [10 — Restricted editing](./10-restricted-editing.md)
- [11 — Widgets](./11-widgets.md)
- [24 — Data format](./24-data-format.md)
- [38 — Build & bundling](./38-build-bundling.md)

Repo references:

- `frontend/apps/web/src/features/documents/mddm-editor/engine/` — current
  IR + emitters + goldens. Target of this refactor.
- `apps/ck5-studio/src/lib/editorConfig.ts` — pinned CK5 v48 plugin list.
