---
title: DOCX export
status: draft
area: export
---

# 31 — DOCX export

## Recommended design

MetalDocs is GPL-only on CK5 (see [01 — license](./01-license-gpl.md)); the
premium **Export to Word** plugin is *not* available to us. We build a
comparable open-source emitter on top of the `docx` npm library, driven by
the CK5 HTML that is the document's single source of truth
(see [25 — MDDM IR bridge](./25-mddm-ir-bridge.md)).

Pipeline:

```
CK5 HTML (editor.getData())
  └─ htmlToExportTree()          ← linkedom parse, thin mapping
       └─ ExportNode tree (ephemeral)
            └─ docx-emitter visitor (existing)
                 └─ docx npm library → .docx bytes
```

Key points:

- **No HTMLtoDOCX.** We already depend on `docx` directly in the emitter; the
  HTMLtoDOCX path is retired. It also cannot render our tables because
  CKEditor wraps every table in `<figure class="table">`, producing a
  nesting HTMLtoDOCX mis-handles (memory: nested-figure-table bug).
- **No MDDM IR at export time.** The docx-emitter visitor is re-pointed from
  the old layout-IR to `ExportNode`, as described in §4–§5 of page 25.
- **Server-side primary, client-side opportunistic.** Node-side export is
  the default (auth, large docs, asset fetch); a browser-side path using the
  same modules is available for quick downloads.
- **Ingress has already canonicalized the HTML** (page 25 §10), so the
  parser is dumb-and-fast and does not re-validate schema.

---

## 1. Architecture

One ephemeral tree between HTML and the `docx` AST. The visitor layer is
unchanged from the existing `frontend/apps/web/src/features/documents/
mddm-editor/engine/docx-emitter/` — only its inputs change.

```
HTML string
   │
   ▼
[htmlToExportTree]      linkedom DOM walk, MDDM-aware
   │
   ▼
ExportNode (root)
   │
   ▼
[docxEmitter.visit]     existing visitor pattern, goldens
   │
   ▼
docx.Document → Packer.toBuffer() → .docx
```

The tree is computed fresh every export. It is not persisted, not cached,
not indexed. See page 25 §4 for the rationale ("disposable visitor input").

---

## 2. Parser layer — `htmlToExportTree`

Thin mapping from CK5 data-view shapes to `ExportNode`s. Lives in
`engine/docx-emitter/html-to-export-tree.ts` (new).

### Shape mapping

| CK5 HTML shape                                                | ExportNode                                                |
|---------------------------------------------------------------|-----------------------------------------------------------|
| `<section class="mddm-section" data-mddm-variant="…">`        | `section` with `variant`, header child, body children     |
| `<ol class="mddm-repeatable" data-mddm-id="…">`               | `repeatable`, `items[]`                                   |
| `<li>` inside `mddm-repeatable`                               | `repeatableItem`, body children                           |
| `<figure class="table"><table data-mddm-variant="…">…</table>`| `table` (flat; `variant` = fixed \| dynamic)              |
| `<tr>`, `<th>`, `<td>`                                        | `tableRow`, `tableCell` (with `isHeader`)                 |
| `<span class="mddm-field" data-mddm-field-id data-mddm-field-type>value</span>` | `field` (inline) with `id`, `type`, `value`    |
| `<div class="mddm-rich-block">`                               | unwrap — emit children only                               |
| `<span class="restricted-editing-exception">`                 | unwrap — exception chrome is not exported                 |
| `<div class="restricted-editing-exception">`                  | unwrap — same                                             |
| `<h1>`–`<h6>`                                                 | `heading` with `level`                                    |
| `<p>`                                                         | `paragraph` (+ `align` from `style="text-align:…"`)       |
| `<ul>` / `<ol>` / `<li>`                                      | `list` (`ordered`), `listItem`                            |
| `<img>`                                                       | `image` with `src`, `alt`, `width`, `height`              |
| `<a href="…">`                                                | `hyperlink`                                               |
| `<strong>` / `<b>`, `<em>` / `<i>`, `<u>`, `<s>`              | inline marks on `text`                                    |
| `<br>`                                                        | `lineBreak`                                               |
| `<blockquote>`                                                | `blockquote`                                              |
| `<figure><figcaption>…</figcaption></figure>` (non-table)     | `figure` with caption child                               |

Widget attributes (`data-mddm-*`) are preserved onto their `ExportNode` —
see [11 — Widgets](./11-widgets.md) and the mapping pages
[19](./19-sections-mapping.md), [20](./20-repeatables-mapping.md),
[21](./21-datatable-mapping.md), [22](./22-fieldgroup-mapping.md),
[23](./23-rich-block-mapping.md).

### Type sketch

```ts
// engine/docx-emitter/export-node.ts
export type ExportNode =
  | { kind: 'root'; children: ExportNode[] }
  | { kind: 'section'; variant: string; header?: ExportNode; body: ExportNode[] }
  | { kind: 'repeatable'; id: string; items: ExportNode[] }
  | { kind: 'repeatableItem'; body: ExportNode[] }
  | { kind: 'heading'; level: 1|2|3|4|5|6; children: Inline[] }
  | { kind: 'paragraph'; align?: 'left'|'center'|'right'|'justify'; children: Inline[] }
  | { kind: 'list'; ordered: boolean; items: ExportNode[] }
  | { kind: 'listItem'; children: ExportNode[] }
  | { kind: 'table'; variant: 'fixed'|'dynamic'; rows: TableRow[] }
  | { kind: 'blockquote'; children: ExportNode[] }
  | { kind: 'figure'; body: ExportNode[]; caption?: string }
  | Inline;

export type Inline =
  | { kind: 'text'; value: string; marks?: Marks }
  | { kind: 'hyperlink'; href: string; children: Inline[] }
  | { kind: 'image'; src: string; alt?: string; width?: number; height?: number }
  | { kind: 'field'; id: string; type: string; value: string }
  | { kind: 'lineBreak' };

export interface Marks {
  bold?: boolean; italic?: boolean; underline?: boolean;
  strike?: boolean; color?: string; size?: number;
}

export interface TableRow { cells: TableCell[] }
export interface TableCell { isHeader: boolean; children: ExportNode[] }
```

### Parser skeleton

```ts
import { parseHTML } from 'linkedom';

export function htmlToExportTree(html: string): ExportNode {
  const { document } = parseHTML(`<div id="__root">${html}</div>`);
  return { kind: 'root', children: walkChildren(document.getElementById('__root')!) };
}

function walkChildren(el: Element): ExportNode[] {
  return Array.from(el.childNodes).flatMap(walk);
}

function walk(node: Node): ExportNode[] {
  if (node.nodeType === 3) return [{ kind: 'text', value: node.textContent ?? '' }];
  if (node.nodeType !== 1) return [];
  const el = node as Element;
  // Unwrap restricted-editing chrome.
  if (el.classList?.contains('restricted-editing-exception')) return walkChildren(el);
  // MDDM widgets first.
  if (el.matches('section.mddm-section')) return [toSection(el)];
  if (el.matches('ol.mddm-repeatable')) return [toRepeatable(el)];
  if (el.matches('figure.table')) return [toTable(el)];
  if (el.matches('span.mddm-field')) return [toField(el)];
  if (el.matches('div.mddm-rich-block')) return walkChildren(el);
  // Standard HTML.
  return [toStandard(el)];
}
```

`toSection`, `toRepeatable`, `toTable`, `toField`, `toStandard` are small
one-shape-each functions; see the mapping pages for attribute contracts.

---

## 3. Emitter layer

The existing `docx-emitter` visitor stays. It already walks a tree of nodes
and emits `docx` AST (`Document`, `Paragraph`, `TextRun`, `Table`, `Row`,
`Cell`, `ImageRun`, `Numbering`). Only the `visit` dispatcher's input type
changes from the old layout-IR to `ExportNode`.

Skeleton (condensed from current code):

```ts
import { Document, Paragraph, TextRun, HeadingLevel, Table, TableRow, TableCell,
         ImageRun, AlignmentType, Packer } from 'docx';

export async function emitDocx(root: ExportNode, assets: AssetResolver) {
  const doc = new Document({
    numbering: buildNumberingStyles(root),
    sections: [{ properties: {}, children: visitBlocks(root, assets) }],
  });
  return Packer.toBuffer(doc);
}

function visitBlocks(node: ExportNode, assets: AssetResolver): (Paragraph | Table)[] {
  switch (node.kind) {
    case 'heading':    return [emitHeading(node)];
    case 'paragraph':  return [emitParagraph(node)];
    case 'list':       return emitList(node);
    case 'table':      return [emitTable(node)];
    case 'section':    return [...visitBlocks(node.header!, assets),
                               ...node.body.flatMap(b => visitBlocks(b, assets))];
    case 'repeatable': return node.items.flatMap(i => emitRepeatableItem(i));
    /* … */
  }
}
```

Goldens, asset-resolver, and ir-hash-free tree-walk all survive.

---

## 4. Parity matrix vs premium **Export to Word**

CKEditor's premium plugin (reference docs:
`https://ckeditor.com/docs/ckeditor5/latest/features/converters/export-word.html`)
is GPL-incompatible and off-limits (page 01). We mirror its observable
output surface where possible:

| Feature (premium export-word)                    | Our emitter                                       | Notes                                         |
|--------------------------------------------------|---------------------------------------------------|-----------------------------------------------|
| Headings `<h1..h6>` → Word heading styles        | ✅ `HeadingLevel.HEADING_1..6`                    | Style names match Word defaults               |
| Paragraph alignment                               | ✅ `AlignmentType.{LEFT,CENTER,RIGHT,JUSTIFIED}` | Read from inline `text-align`                 |
| Ordered/unordered lists                           | ✅ `numPr` with our numbering style               | One `Numbering` entry per list variant        |
| Nested lists                                      | ✅ level in `numPr`                               |                                               |
| Tables (flat)                                     | ✅ `Table` + `TableRow`/`TableCell`               | Nested tables rejected at ingress (page 13)   |
| Table headers                                     | ✅ cell shading + bold run                        |                                               |
| Images (embedded)                                 | ✅ `ImageRun` via `asset-resolver`                | Inline bytes                                  |
| Images (linked)                                   | ⚠ uncertain — not yet wired                       | Defer                                         |
| Bold / italic / underline / strike                | ✅ `TextRun` options                              |                                               |
| Font color / size                                 | ✅ `TextRun` `color`/`size`                       |                                               |
| Hyperlinks                                        | ✅ `ExternalHyperlink`                            |                                               |
| Blockquote                                        | ✅ paragraph with "Quote" style                   |                                               |
| Line breaks                                       | ✅ `TextRun.break`                                |                                               |
| Captions (figure + figcaption)                    | ✅ paragraph with "Caption" style                 |                                               |
| Page breaks                                       | ✅ `PageBreak` run where `mddm-page-break` marker | Optional per template                         |
| Headers/footers                                   | ⚠ uncertain — template-driven only                | Deferred to template engine                   |
| Table of contents                                 | ❌ not in MVP                                     |                                               |
| Track changes                                     | ❌ not supported                                  |                                               |
| Comments                                          | ❌ not supported                                  |                                               |
| **SDT / content controls (fields)**               | ❌ see §5 below                                   | `docx` library SDT support ⚠ uncertain        |

### Import-from-Word parity reference

The inverse premium plugin
(`https://ckeditor.com/docs/ckeditor5/latest/features/converters/import-word/import-word.html`)
informs which Word features we must *emit cleanly enough* for a lossless
round-trip if that path ever matters. Scope: headings, paragraphs, lists,
tables, images, basic runs. Nothing exotic. Not a current requirement.

---

## 5. Known limitation vs premium — no SDT

Premium Export to Word can emit **Structured Document Tags (SDT /
content controls)** to represent fields with a stable Word-side identity.
We cannot: the `docx` npm library does not publicly document SDT support.
⚠ uncertain — whether a low-level escape exists; if so we would add SDT
for `field` nodes in a follow-up.

Current design: **each `field` node is emitted as a plain `TextRun`
carrying the filled value inline**. Acceptable because:

- Round-trip authority is CK5 HTML, not the DOCX (page 25 §1).
- DOCX is an export artefact, not a source of truth.
- Users opening the DOCX see the filled text, not a placeholder control.

If we later need SDT, swap the `field` emitter only.

---

## 6. Repeatables

Emit our own numbering style per repeatable variant (`numbering` reference
registered once on the `Document`). Each item is flattened to one or more
paragraphs that share `numPr`:

```ts
function emitRepeatableItem(item: ExportNode & { kind: 'repeatableItem' }) {
  const [first, ...rest] = item.body.flatMap(b => visitBlocks(b, assets));
  first.numbering = { reference: 'mddm-repeatable', level: 0 };
  return [first, ...rest];
}
```

See [20 — Repeatables mapping](./20-repeatables-mapping.md) for the HTML
contract.

---

## 7. Sections

Per-section handling is config-driven by `variant` (see
[19 — Sections mapping](./19-sections-mapping.md)):

- **Header.** Emit as `Heading 1` or `Heading 2` depending on the variant's
  `headerLevel`.
- **Body.** Emit in document order as normal paragraphs / lists / tables.
- **Page break.** If the variant declares `breakBefore`, prepend a
  `Paragraph` with a `PageBreak` run.

No section-level Word "section" construct is used; our sections are
semantic MDDM groupings, not Word sections.

---

## 8. Tables — fixed vs dynamic

`mddm-table-variant` is a runtime (editor) concern. In DOCX both emit as
ordinary `Table` structures; no lock/unlock metadata is persisted. Opening
the DOCX in Word, every cell is editable regardless of the variant the
template author chose.

⚠ uncertain — whether we want to reflect the fixed variant as a Word SDT
wrapping locked cells. Defer with §5.

See [21 — Datatable mapping](./21-datatable-mapping.md) and the memory
note that CK5 wraps tables in `<figure class="table">` (handled in §2;
no nested tables ever reach the emitter).

---

## 9. Assets / images

The existing `asset-resolver` at
`engine/docx-emitter/asset-resolver.ts` stays as-is:

- Input: `image.src` (can be absolute URL, relative URL, or MDDM asset URI).
- Output: `Buffer` of the image bytes + inferred `transformation` size.
- Emitter: `new ImageRun({ data, transformation: { width, height } })`.

Remote fetches run server-side only; in the browser-side path the resolver
uses `fetch` + `arrayBuffer()`.

---

## 10. Golden tests

Current goldens live in `engine/docx-emitter/__fixtures__/` and compare the
raw `docx` AST (not bytes) produced from **layout-IR** inputs.

Migration plan:

1. **Re-anchor on HTML.** For each golden, replace its IR input with the
   equivalent CK5 HTML fixture. The expected `docx` output stays byte-identical.
2. **Bridging harness.** For one PR, run every golden twice — through the
   old IR path and the new HTML → ExportNode path — and assert both produce
   the same `docx` AST. This proves parity before deleting the IR path.
3. **Retire IR-only goldens case-by-case.** Some goldens depend on fields
   that no longer exist in HTML round-trip (see page 25 §5). Retire with
   PR notes, do not silently delete.

See [34 — Golden tests](./34-golden-tests.md) for the harness.

---

## 11. Server-side vs client-side

| Path          | When                            | Modules used                              |
|---------------|---------------------------------|-------------------------------------------|
| Server (Node) | Default; large docs, auth-gated | `linkedom`, `docx`, `asset-resolver`      |
| Client (browser) | "Quick download" UX          | `linkedom` (or native `DOMParser`), `docx`|

Both paths share the parser and the emitter. The browser-side path uses
`fetch` for asset resolution; the server-side path uses the server-side
resolver (which may hit S3, DB, etc.).

---

## 12. Failure modes

- **Invalid HTML.** Should never reach the emitter: ingress canonicalization
  (page 25 §10) strips non-conforming structure. If it does, the parser
  emits a lossy best-effort tree and logs a `drift` event.
- **Unsupported elements (GHS leakage).** Elements preserved by General
  HTML Support but not in our mapping table are dropped by `toStandard`
  with a `console.warn`. See [18 — HTML support](./18-html-support.md).
- **Nested tables.** Rejected at ingress (page 13); never reach the
  emitter. The `<figure class="table">` wrapper is the only wrapping we
  expect; nested occurrences are logged and flattened.
- **Missing asset.** `asset-resolver` returns `null`; the emitter skips
  the image and inserts a `[image missing]` TextRun with a warning style.
- **Huge tables.** `docx` has no documented upper bound; ⚠ uncertain
  beyond ~5k rows. Benchmark before promising.

---

## Open questions

- ⚠ uncertain — whether `docx` exposes any SDT / content-control escape
  hatch. If yes, fields should upgrade to SDT for Word-side identity.
- ⚠ uncertain — how many existing goldens survive the IR → HTML re-anchor
  byte-for-byte (audit tracked in page 25 §5).
- ⚠ uncertain — headers/footers story. Likely lands with the template
  engine, not this page.
- ⚠ uncertain — browser-side asset resolution performance for docs with
  many remote images; may need chunking.

---

## Sources & cross-refs

- [01 — License (GPL)](./01-license-gpl.md) — premium plugin is off-limits
- [11 — Widgets](./11-widgets.md)
- [19 — Sections mapping](./19-sections-mapping.md)
- [20 — Repeatables mapping](./20-repeatables-mapping.md)
- [21 — Datatable mapping](./21-datatable-mapping.md)
- [22 — Fieldgroup mapping](./22-fieldgroup-mapping.md)
- [23 — Rich block mapping](./23-rich-block-mapping.md)
- [24 — Data format](./24-data-format.md)
- [25 — MDDM IR bridge](./25-mddm-ir-bridge.md)
- [32 — PDF export](./32-pdf-export.md)
- [34 — Golden tests](./34-golden-tests.md)

External (reference only, GPL-incompatible — do **not** copy code):

- CKEditor Export to Word — feature surface & expected DOCX shape:
  `https://ckeditor.com/docs/ckeditor5/latest/features/converters/export-word.html`
- CKEditor Import from Word — inverse mapping, informs round-trip goals:
  `https://ckeditor.com/docs/ckeditor5/latest/features/converters/import-word/import-word.html`

Repo references:

- `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/`
  — existing visitor, asset-resolver, goldens. Input adapter is the only
  file that changes.
- `docx` npm package — pinned; the only DOCX writer we use.
