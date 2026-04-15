---
title: PDF export
status: draft
area: export
---

# 32 — PDF export

MetalDocs runs CKEditor 5 under the **GPL tier** (see
[01 — license](./01-license-gpl.md)). That disqualifies both the premium
**Export to PDF** plugin and the premium **Pagination** plugin. This page
describes the in-house path: serialize CK5 HTML (see
[24 — Data format](./24-data-format.md)), wrap it in a print-ready HTML
shell, apply a CSS Paged Media stylesheet, and render with headless
Chromium. The same HTML is the canonical storage artefact (see
[25 — MDDM IR bridge](./25-mddm-ir-bridge.md)); PDF is always regenerated
from it on demand.

---

## Recommended design

1. **Server-side headless Chromium + CSS Paged Media is the default path.**
   CK5's data view HTML is wrapped in an HTML shell with a print
   stylesheet, POSTed to a renderer service (Gotenberg or a Puppeteer /
   Playwright worker), and returned as a PDF byte stream. Chromium's
   layout engine is the same one the author sees in the editor preview —
   this is how we achieve visual parity without the premium plugin.
2. **Client-side `window.print()` is a secondary convenience** for quick
   previews and offline use. It reuses the exact same print stylesheet,
   rendered into a hidden iframe populated with `editor.getData()`.
3. **Pagination is pure CSS Paged Media.** `@page` rules + `break-*`
   properties + `thead { display: table-header-group; }` cover the common
   cases. For footnotes, cross-reference page numbers, or multi-column
   layouts that Chromium does not implement natively, we layer
   **Paged.js** (MIT) — it polyfills the missing CSS Paged Media spec in
   the browser so the in-browser and headless renderings stay byte-close.
4. **PDF is never the source of truth.** It is a derivative. Regenerate
   on every export; do not version.

Current implementation lives at
`frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-pdf.ts`
and composes the shell via
`engine/print-stylesheet/wrap-print-document.ts` + `print-css.ts`. The
server component today is a Gotenberg endpoint at
`/api/v1/documents/:id/render/pdf`.

---

## 1. Input contract

The exporter consumes **CK5 data-view HTML** — the string returned by
`editor.getData()`, already stripped of widget chrome (see
[24 §3](./24-data-format.md)). The renderer does not re-parse the model
and does not run CK5 server-side.

Wrap the HTML in a minimal shell before handing it to Chromium:

```ts
function wrapInPrintDocument(bodyHtml: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<title>MDDM Document</title>
<link rel="stylesheet" href="print.css" />
</head>
<body>
${bodyHtml}
</body>
</html>`;
}
```

Two delivery options:

- **Inline `<style>`** — stylesheet interpolated into the document
  (current implementation, see `wrap-print-document.ts`). Simplest; one
  HTTP payload.
- **Linked `<link rel="stylesheet">`** — stylesheet uploaded as a
  separate multipart part (current Gotenberg path, see `export-pdf.ts`
  lines 84–86). Keeps CSS cacheable and inspectable but requires the
  renderer to accept multipart form data.

Either is acceptable. Prefer linked when the renderer supports it.

---

## 2. Print stylesheet skeleton

The heart of pagination without the premium plugin is `@page`. Minimal
skeleton derived from `print-css.ts`:

```css
@page {
  size: A4;
  margin: 25mm;

  @top-center    { content: element(running-header); }
  @bottom-right  { content: counter(page) " / " counter(pages); }
}

@page :first  { @top-center { content: none; } }
@page :left   { @top-left    { content: element(running-header); } }
@page :right  { @top-right   { content: element(running-header); } }

html, body { margin: 0; padding: 0; font-family: "Inter", sans-serif; }

/* Running header / footer source elements. */
.mddm-running-header { position: running(running-header); }

/* Table header repeat on every page. */
thead { display: table-header-group; }
tfoot { display: table-footer-group; }

/* Section breaks + keep-together rules. */
.mddm-section            { break-before: page; }
.mddm-section-header,
.mddm-field-group,
.mddm-field              { break-inside: avoid; }
h1, h2, h3               { break-after: avoid; }

/* Widow / orphan control for prose. */
p { orphans: 3; widows: 3; }

/* Hide any editor chrome that might leak through. */
.ck-widget__selection-handle,
.ck-widget__resizer,
.ck-placeholder,
.bn-side-menu,
.bn-drag-handle          { display: none !important; }

/* Force background colours/images into the PDF. */
@media print {
  * { -webkit-print-color-adjust: exact !important;
      color-adjust: exact !important; }
}
```

Key points:

- **`size: A4` + `preferCSSPageSize: true`** in the Puppeteer call
  (§10) lets CSS own the page geometry.
- **`@page :first`, `:left`, `:right`** handle cover pages and
  left/right-hand running chrome.
- **`break-before: page` on `.mddm-section`** replaces the premium
  pagination plugin's section boundary.
- **`break-inside: avoid`** on headers, field groups, small tables
  keeps atomic regions together.
- **Chrome suppression** — see §5.

---

## 3. Pagination without the premium plugin

Native CSS Paged Media, as implemented by Chromium, already supports:

- `@page` with margin boxes (`@top-*`, `@bottom-*`)
- `break-before`, `break-after`, `break-inside`
- `orphans`, `widows`
- `counter(page)`, `counter(pages)` in margin boxes
- `display: table-header-group` repeating `<thead>` across pages

Chromium does **not** natively implement:

- `string-set` / `string()` (running strings from content)
- `target-counter()` for cross-reference page numbers
- `@footnote` / `float: footnote`
- Named pages (`page: cover`)
- Some `running()` + `position: running()` edge cases for complex
  layouts

### Paged.js fallback

When a layout needs any unsupported feature, load **Paged.js** (MIT, open
source) in the wrapped HTML:

```html
<script src="/vendor/paged.polyfill.js"></script>
```

Paged.js polyfills the missing CSS Paged Media Level 3 features. The same
polyfilled HTML renders identically in the browser (`window.print()`) and
in headless Chromium, which keeps preview and export aligned.

Cost: Paged.js runs a JS pass that splits the DOM into page boxes before
print. On documents >50 pages this adds noticeable latency (⚠ uncertain
— benchmark on representative templates; if native CSS suffices for a
document class, skip Paged.js and keep rendering fast).

---

## 4. Widget chrome exclusion

Per [24 §3](./24-data-format.md), `editor.getData()` already excludes
editing-view chrome: selection handles, fake-selection wrappers,
`contenteditable` attributes, placeholder text. The stored HTML is
clean.

Two leak paths still to guard:

1. **Fill-mode preview exports.** If someone exports while the editor
   is live and mistakenly passes the editing DOM instead of
   `getData()`, chrome classes like `.ck-widget__selection-handle`,
   `.ck-widget__resizer`, `.ck-placeholder` appear. The print stylesheet
   hides them defensively (§2).
2. **Restricted-editing exception outlines.** The default stylesheet
   shows a dashed outline around `.restricted-editing-exception`. That
   outline is editing-view CSS; it must not be carried into
   `print-css.ts`.

Rule: the print stylesheet is a **standalone** file. Never import the
editor's CSS into it. Start from the skeleton in §2 and add only what
the paginated output needs.

---

## 5. Headers and footers

Two approaches, both open-source:

### (a) CSS `position: running()`

```html
<div class="mddm-running-header">
  Acme Corp — Purchase Order #<span data-field="po-number"></span>
</div>
```

```css
.mddm-running-header { position: running(running-header); }
@page { @top-center { content: element(running-header); } }
```

Chromium supports this with `preferCSSPageSize`, though edge cases on
first/left/right pages may need Paged.js (⚠ uncertain — verify per
Chromium version).

### (b) Paged.js `string-set`

```css
h1 { string-set: chapter content(); }
@page { @top-left { content: string(chapter); } }
```

Lets the header track the current chapter title. Requires Paged.js.

---

## 6. Page numbers

Pure CSS, no polyfill needed:

```css
@page {
  @bottom-right { content: counter(page) " / " counter(pages); }
}
@page :first {
  @bottom-right { content: none; }   /* suppress on cover */
}
```

Cross-references ("see page 12") need `target-counter(url(#anchor),
page)`, which is **not** in Chromium today. Use Paged.js or a JS
pre-pass that walks `<a href="#id">` nodes and substitutes the resolved
page number (⚠ uncertain — premium Export-to-PDF handles this out of
the box; our parity is "works via Paged.js, not via raw Chromium").

---

## 7. Images

Image `src` in the saved HTML typically points at `/api/images/...`
(auth-bound). A headless renderer cannot reach that URL without
credentials.

Strategy, as implemented in
`engine/export/inline-asset-rewriter.ts`:

1. Extract every `<img src>` from the body HTML.
2. Resolve each asset via `AssetResolver` (honouring ceilings in
   `RESOURCE_CEILINGS` — max images per document, max total bytes).
3. Rewrite each `src` to a `data:` URI (base64) before sending to the
   renderer.

The renderer then runs fully offline and deterministic. CDN-backed
absolute URLs are a secondary option when size budgets forbid inlining,
but require the renderer host to have outbound network access.

---

## 8. Fonts

Same sandboxing principle as images. Do **not** rely on a Google Fonts
`<link>` — headless workers are often network-restricted, and even when
they aren't the fetch adds variable latency and risks missing glyphs.

Self-host via `@font-face` in the print stylesheet:

```css
@font-face {
  font-family: "Inter";
  src: url(data:font/woff2;base64,…) format("woff2");
  font-weight: 400;
  font-style: normal;
}
```

Or, if the renderer supports multipart uploads (Gotenberg does), ship
the font file as a separate part and reference it by filename.

Always declare `font-synthesis: none` to avoid Chromium's faux-bold /
faux-italic, which differs from the editor's rendering.

---

## 9. Server-side runner

Two concrete implementations:

### (a) Gotenberg (current)

Our existing pipeline (see `export-pdf.ts`) POSTs multipart form data to
`/api/v1/documents/:id/render/pdf`. The server forwards to a Gotenberg
container that runs headless Chromium internally. Pros: off-the-shelf,
Apache-2 licensed, no Node Chromium management. Cons: extra service
hop; limited control over Puppeteer flags.

### (b) Direct Puppeteer / Playwright

```ts
import puppeteer from "puppeteer";

export async function renderPdf(fullHtml: string): Promise<Buffer> {
  const browser = await puppeteer.launch({ headless: "new" });
  try {
    const page = await browser.newPage();
    await page.setContent(fullHtml, { waitUntil: "networkidle0" });
    return await page.pdf({
      format: "A4",
      preferCSSPageSize: true,
      printBackground: true,
      margin: { top: "25mm", right: "25mm", bottom: "25mm", left: "25mm" },
    });
  } finally {
    await browser.close();
  }
}
```

Keep a **warm browser pool** if QPS justifies it; cold-start is ~1–2 s.
Both Puppeteer and Playwright are Apache-2 licensed — no GPL conflict
with the editor (see [01](./01-license-gpl.md)).

---

## 10. Parity matrix vs premium Export-to-PDF

| Premium feature                      | Our GPL equivalent                                | Parity |
|--------------------------------------|---------------------------------------------------|--------|
| Page size / margins                  | `@page { size; margin }`                          | ✔      |
| Running header / footer              | `position: running()` + `@top-*` margin box       | ✔      |
| Page number / page count             | `counter(page)` / `counter(pages)`                | ✔      |
| First-page / left / right variants   | `@page :first`, `:left`, `:right`                 | ✔      |
| Section break before / after         | `break-before: page` on MDDM section wrappers     | ✔      |
| Keep together                        | `break-inside: avoid`                             | ✔      |
| Widow / orphan control               | `orphans`, `widows`                               | ✔      |
| Repeating table headers              | `thead { display: table-header-group }`           | ✔      |
| Fonts                                | `@font-face` self-hosted                          | ✔      |
| Images                               | `data:` URI rewrite via asset resolver            | ✔      |
| Cross-reference page numbers         | `target-counter` (Paged.js) or JS pre-pass        | ⚠ uncertain |
| Table of contents with page numbers  | CSS counters + `target-counter` or JS pre-pass    | ⚠ uncertain |
| Footnotes                            | Paged.js `float: footnote`                        | ⚠ uncertain |
| Multi-column layouts                 | `column-count` + Paged.js page-column break fixes | ⚠ uncertain |
| Watermarks                           | `@page` background or fixed-position element      | ✔      |
| Digital signatures                   | out of scope                                      | ✘      |
| PDF/A compliance                     | Gotenberg flag / Chromium `--pdfa` (if exposed)   | ⚠ uncertain |

---

## 11. Client-side fallback

For quick in-browser preview:

```ts
function clientPrint(editor: Editor) {
  const html = editor.getData();
  const iframe = document.createElement("iframe");
  iframe.style.position = "fixed";
  iframe.style.inset = "0";
  iframe.style.width = "0";
  iframe.style.height = "0";
  iframe.style.border = "0";
  document.body.appendChild(iframe);

  const doc = iframe.contentDocument!;
  doc.open();
  doc.write(wrapInPrintDocument(html));
  doc.close();

  iframe.contentWindow!.focus();
  iframe.contentWindow!.print();
}
```

Uses the exact same print stylesheet and shell as the server path. The
user's browser save-as-PDF dialog produces a file close to the server
output. It is not byte-identical (font substitution, user margin
overrides) and must not be treated as canonical.

---

## 12. Versioning

PDF is never stored as the source of truth. The canonical artefact is
the CK5 HTML (see [25](./25-mddm-ir-bridge.md)). Exports regenerate on
every download. If a customer needs an immutable snapshot of an earlier
version, snapshot the **HTML** at that version and re-render — do not
cache the PDF bytes and serve them back.

Consequence: renderer pinning (`RendererPin` in
`export-pdf.ts`) exists only so historical versions re-render under the
stylesheet they were authored against. It is not a PDF-storage scheme.

---

## Open questions

- ⚠ uncertain — Paged.js latency on 50+ page templates; needs a
  benchmark before we make it the default vs opt-in.
- ⚠ uncertain — exact Chromium version where `position: running()` is
  stable for `@top-left` / `@top-right` variants; pin renderer version
  accordingly.
- ⚠ uncertain — best TOC-with-page-numbers strategy: Paged.js
  `target-counter`, a JS post-pass that walks the rendered page boxes,
  or a two-pass render (draft → read page map → final).
- ⚠ uncertain — whether Gotenberg exposes the Puppeteer flags we need
  for PDF/A and linearization; may require switching to a direct
  Puppeteer worker.
- ⚠ uncertain — memory envelope of a warm Puppeteer pool in production
  vs the per-request Gotenberg model.

---

## Cross-refs

- [01 — License (GPL)](./01-license-gpl.md) — why premium Export-to-PDF
  and Pagination are off-limits.
- [24 — Data format](./24-data-format.md) — `getData()` contract; why
  the stored HTML is already free of editor chrome.
- [25 — MDDM IR bridge](./25-mddm-ir-bridge.md) — HTML as source of
  truth; PDF as a disposable derivative.
- [31 — DOCX export](./31-docx-export.md) — sibling export path,
  different toolchain, same input HTML.

## Repo references

- `frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-pdf.ts`
  — current exporter entry point (Gotenberg multipart path).
- `frontend/apps/web/src/features/documents/mddm-editor/engine/print-stylesheet/`
  — `print-css.ts` and `wrap-print-document.ts`.
- `frontend/apps/web/src/features/documents/mddm-editor/engine/export/inline-asset-rewriter.ts`
  — `<img src>` → `data:` URI rewrite for offline rendering.
