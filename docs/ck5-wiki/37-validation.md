---
title: Server-side validation & sanitization
status: draft
area: backend
priority: HIGH
---

# 37 — Server-side HTML validation & sanitization

CK5's wire format is HTML (see [24 — Data format](./24-data-format.md)) and
our storage contract is "CK5 HTML is source of truth"
(see [25 — MDDM bridge §1](./25-mddm-ir-bridge.md)). That means every byte
of content we persist arrived from a browser-side editor over an HTTP save
call. This page covers how the backend validates and sanitizes that HTML
before it reaches the database or an export worker.

---

## Recommended design

1. **Sanitize with `sanitize-html`** on the save path. Allowlist-based;
   mirrors the CK5 schema + MDDM element set; drops everything else
   silently (with a log line).
2. **Structurally validate** the sanitized output with a `linkedom` walk:
   one-of-each header/body per section, repeatable children shape, field
   span constraints, no nested tables.
3. **Reject on structural failure**, return `{ path, rule, message }[]` so
   the client can highlight offending nodes.
4. **Canonicalize on save (not autosave)** via a one-shot headless CK5
   `setData → getData`. Stabilizes whitespace/attribute order; detects
   silent-drop drift as a side effect (see
   [25 — §10](./25-mddm-ir-bridge.md#10-validation-and-drift)).
5. **Never trust the client.** CK5's schema filter is the first gate, not
   the last. A tampered client can emit arbitrary HTML; assume it will.

---

## 1. Why validate server-side

The CK5 editor enforces its schema on `setData`/`getData`, but that
enforcement only binds **honest clients**. The attacker model we care about:

- **Compromised client** — CSP bypass, XSS in a third-party dep, malicious
  browser extension. Any of these can call our save endpoint with raw HTML.
- **Buggy client** — a half-shipped converter, a stale bundle, a GHS
  regression. These emit valid-looking but schema-violating HTML.
- **Replay / direct API use** — anyone with auth can POST HTML directly.

So the editor's schema is the **first** gate (UX + happy path). The server
is the **last** gate (security + invariants). They are not redundant;
they protect different threat boundaries.

⚠ uncertain — whether we expose a raw-HTML import endpoint
(Word paste, legacy doc migration). If yes, it runs through the same
sanitizer + validator.

---

## 2. Sanitizer library choice

Two credible options in Node:

| Library         | License | XSS record | Config ergonomics | Notes                              |
|-----------------|---------|------------|-------------------|------------------------------------|
| `sanitize-html` | MIT     | Good       | Declarative JSON  | Node-native; deep CK5 ecosystem fit|
| `DOMPurify`     | MPL/Apache | Excellent | Hook-based      | Needs `linkedom`/`jsdom` in Node   |

**Pick `sanitize-html`** for MetalDocs. Reasons:

- Config is a single JSON object — reviewable in PR, easy to diff.
- Node-native; no DOM shim required.
- Our allowlist is large but static; we do not need DOMPurify's
  per-element hooks.

Keep DOMPurify as a **fallback** if we ever need stricter attribute-value
enforcement (e.g. CSS-in-style parsing). Its XSS record is the strongest
in the ecosystem and worth the extra shim cost if we need it.

⚠ uncertain — whether sanitize-html's CSS property allowlist is sufficient
for our print-stylesheet overrides. Benchmark on a real template before
committing the `style` whitelist (§5).

---

## 3. Allowlist schema

Mirrors the CK5 editor schema (see [04](./04-schema.md)) plus MDDM custom
elements from [19–23]. Anything else is dropped.

### Standard HTML

```
p, h1, h2, h3, h4, h5, h6,
ul, ol, li,
strong, em, u, s,
a[href, title, target, rel],
br, hr, blockquote,
figure, figcaption,
table, thead, tbody, tr, th, td, caption,
img[src, alt, width, height]
```

### MDDM custom

| Element   | Allowed attributes                                                      | Allowed classes                                                                                    |
|-----------|-------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------|
| `section` | `class`, `data-section-id`, `data-variant`                              | `mddm-section`                                                                                     |
| `header`  | `class`                                                                 | `mddm-section__header`                                                                             |
| `div`     | `class`, `data-repeatable-id`, `data-field-group-id`, `data-mddm-schema`| `mddm-section__body`, `mddm-rich-block`, `mddm-field-group`                                        |
| `ol`      | `class`, `data-repeatable-id`                                           | `mddm-repeatable`                                                                                  |
| `li`      | `class`, `data-item-id`                                                 | `mddm-repeatable__item`                                                                            |
| `span`    | `class`, `data-field-id`, `data-field-type`, `data-field-label`, `data-field-required` | `mddm-field`, `restricted-editing-exception`                                        |

**Class allowlist is exact**: a `class="mddm-section foo"` attribute drops
`foo` and keeps `mddm-section`. No wildcards, no prefix matches, no
inline-authored classes.

---

## 4. `data-*` attributes

sanitize-html's default is wildcard `data-*` matching. **Turn that off**
and enumerate per element (table above). Rationale: a stray
`data-onload-trick` is harmless today, but an attacker who finds a browser
quirk that upcasts `data-*` into event handlers gets a free XSS. Narrow
surface; fewer footguns.

Config snippet:

```ts
allowedAttributes: {
  section: ['class', 'data-section-id', 'data-variant'],
  span:    ['class', 'data-field-id', 'data-field-type',
            'data-field-label', 'data-field-required'],
  // ...per element, never '*'
}
```

---

## 5. Globally banned attributes

Regardless of element:

- `style` — **banned by default**. If a specific need emerges, allowlist
  a named set of safe properties only (`text-align`, `color`,
  `background-color`, `font-weight`, `font-style`, `text-decoration`).
  No `url(…)`, no `expression(…)`, no `position:fixed`.
- `on*` (all event handlers) — banned.
- `srcdoc`, `sandbox` on `iframe` — `iframe` is not in the allowlist at
  all, so this is belt-and-suspenders.
- `formaction`, `xlink:href` — banned.
- Any attribute whose value matches `/^javascript:/i` after trimming and
  entity-decoding — dropped even if the attribute name is allowed.

⚠ uncertain — whether we allow `style` on `td` for column-width overrides.
Prefer `<col>` + stylesheet. Resolve when table-authoring doc lands.

---

## 6. URL validation

Applies to `href` and `src`. Allowed schemes:

- `http:`, `https:`
- `mailto:`
- Relative URLs (`./foo`, `/foo`, `foo`)
- Our CDN origin (`https://cdn.metaldocs.example/...`) — enforced as a
  hostname allowlist, not a prefix match.

Banned:

- `javascript:`, `vbscript:`, `data:` (except see below), `file:`,
  unknown schemes.
- `data:` URLs on `img[src]` — allowed only for `image/png`, `image/jpeg`,
  `image/gif`, `image/webp`. `image/svg+xml` is **banned** (SVG can carry
  script).

Normalization: decode percent-encoding and HTML entities before the
scheme check. `&#106;avascript:` must be caught.

---

## 7. Structural validation

Post-sanitize, walk the DOM with `linkedom` and enforce MDDM invariants
that sanitize-html cannot express:

1. Every `<section class="mddm-section">` has **exactly one**
   `<header class="mddm-section__header">` child and **exactly one**
   `<div class="mddm-section__body">` child — in that order.
2. Every `<ol class="mddm-repeatable">` has **only**
   `<li class="mddm-repeatable__item">` children (no stray `<li>`,
   no nested repeatables).
3. No `<table>` inside `<td>` — nested tables break DOCX export.
4. Every `<span class="mddm-field">` has non-empty `data-field-id`,
   a `data-field-type` matching a known enum, and exactly one text-node
   child (no nested elements).
5. Every element bearing `class="restricted-editing-exception"` has a
   non-empty text range (empty exceptions break caret navigation,
   see [10 — Restricted editing](./10-restricted-editing.md)).
6. `data-mddm-schema` on the root matches a known version.

### Code sketch

```ts
import { parseHTML } from 'linkedom';
import sanitizeHtml from 'sanitize-html';

export interface ValidationError {
  path: string;   // CSS-ish selector to the offending node
  rule: string;   // e.g. 'mddm-section.exactly-one-header'
  message: string;
}

export function validateSaveHtml(raw: string): {
  html: string;
  errors: ValidationError[];
} {
  const clean = sanitizeHtml(raw, SANITIZE_CONFIG);
  const { document } = parseHTML(`<root>${clean}</root>`);
  const errors: ValidationError[] = [];

  for (const section of document.querySelectorAll('section.mddm-section')) {
    const headers = section.querySelectorAll(':scope > header.mddm-section__header');
    const bodies  = section.querySelectorAll(':scope > div.mddm-section__body');
    if (headers.length !== 1) errors.push({
      path: pathOf(section),
      rule: 'mddm-section.exactly-one-header',
      message: `expected 1 header, found ${headers.length}`,
    });
    if (bodies.length !== 1) errors.push({
      path: pathOf(section),
      rule: 'mddm-section.exactly-one-body',
      message: `expected 1 body, found ${bodies.length}`,
    });
  }

  for (const ol of document.querySelectorAll('ol.mddm-repeatable')) {
    for (const child of ol.children) {
      if (child.tagName !== 'LI' ||
          !child.classList.contains('mddm-repeatable__item')) {
        errors.push({
          path: pathOf(child),
          rule: 'mddm-repeatable.only-items',
          message: `unexpected child <${child.tagName.toLowerCase()}>`,
        });
      }
    }
  }

  // ...rules 3–6 follow the same shape.

  return { html: clean, errors };
}
```

`pathOf(node)` emits a stable CSS-ish selector
(`section[data-section-id=abc] > div.mddm-section__body > ol:nth-child(2)`)
so the client can scroll the editor selection to it.

### sanitize-html config skeleton

```ts
import type { IOptions } from 'sanitize-html';

export const SANITIZE_CONFIG: IOptions = {
  allowedTags: [
    'p', 'h1','h2','h3','h4','h5','h6',
    'ul','ol','li','strong','em','u','s','a',
    'br','hr','blockquote',
    'figure','figcaption',
    'table','thead','tbody','tr','th','td','caption',
    'img',
    'section','header','div','span',
  ],
  allowedAttributes: {
    a:       ['href','title','target','rel'],
    img:     ['src','alt','width','height'],
    section: ['class','data-section-id','data-variant'],
    header:  ['class'],
    div:     ['class','data-repeatable-id','data-field-group-id','data-mddm-schema'],
    ol:      ['class','data-repeatable-id'],
    li:      ['class','data-item-id'],
    span:    ['class','data-field-id','data-field-type',
              'data-field-label','data-field-required'],
    // no '*' wildcard, no style anywhere.
  },
  allowedClasses: {
    section: ['mddm-section'],
    header:  ['mddm-section__header'],
    div:     ['mddm-section__body','mddm-rich-block','mddm-field-group'],
    ol:      ['mddm-repeatable'],
    li:      ['mddm-repeatable__item'],
    span:    ['mddm-field','restricted-editing-exception'],
  },
  allowedSchemes: ['http','https','mailto'],
  allowedSchemesByTag: {
    img: ['http','https','data'],
  },
  allowedSchemesAppliedToAttributes: ['href','src'],
  allowProtocolRelative: false,
  disallowedTagsMode: 'discard',
};
```

---

## 8. Error reporting

The save endpoint returns:

```json
{
  "status": "rejected",
  "errors": [
    {
      "path": "section[data-section-id=\"intro\"] > div.body",
      "rule": "mddm-section.exactly-one-body",
      "message": "expected 1 body, found 2"
    }
  ]
}
```

Client-side, the editor maps `path` back to a model position via a CK5
view-to-model lookup and highlights the offending widget. Users never see
"save failed" with no detail — they see the specific node that violates
the rule.

Sanitizer-only drops (e.g. "we stripped a `<script>`") are **logged but
not surfaced as errors**. The save proceeds with the cleaned HTML.
Rationale: sanitizer drops are security wins, not user-correctable
authoring mistakes. Surface them in an admin dashboard instead.

---

## 9. Canonicalization

After sanitize + validate, optionally run the HTML through a one-shot
headless CK5 instance: `setData(html)` → `getData()`. This normalizes:

- Whitespace between block elements
- Attribute order
- Self-closing vs paired tag form
- Boolean attribute form
- Entity encoding

Keeps stored bytes stable across clients and across CK5 patch versions
covered by our pin. Also serves as a **drift detector**: if
`canonicalized !== sanitized`, CK5 either dropped something (log + alert)
or normalized it (expected).

Cost: CK5 cold-start is ~100ms per worker; warm instance is single-digit
ms per doc. Acceptable on the save path; **do not** canonicalize on every
autosave tick — only on final save (see
[26 — Autosave](./26-autosave-persistence.md)).

⚠ uncertain — whether we run canonicalization inline (reject if it drops
content) or async (accept write, compare in background, alert on drift).
Leaning async per [25 — §10](./25-mddm-ir-bridge.md#10-validation-and-drift).

---

## 10. Threat model

### In scope

- **Stored XSS via field values** — `<span class="mddm-field">Acme<script>…</script></span>`. Blocked by sanitizer.
- **Stored XSS via `href="javascript:…"`** on a legitimate-looking link. Blocked by URL validation (§6).
- **Template hijack** — an attacker submits a "template" whose
  restricted-editing exceptions cover the whole document, turning fill
  mode into free editing. Blocked by structural rule 5 + a template-level
  invariant (at least one non-exception region).
- **SVG script injection** via `<img src="data:image/svg+xml,…">`. Blocked
  by banning `image/svg+xml` in `data:` (§6).
- **GHS regression** — a future v48.x changes GHS defaults and lets
  `onclick` through. Caught by the server allowlist regardless.

### Out of scope

- **CSRF** on the save API — handled by auth middleware + SameSite cookies.
- **DoS via huge HTML** — handled by request-size limits upstream. We
  assume the payload fits in memory by the time it reaches the sanitizer.
- **Privilege escalation** (user A saves into user B's document) — handled
  by row-level auth in the backend, not here.

---

## 11. Performance

Rough budget on a representative 50-page template (~500 KB HTML):

| Step                  | Expected cost    |
|-----------------------|------------------|
| sanitize-html         | 5–15 ms          |
| linkedom parse + walk | 10–25 ms         |
| Structural rules      | 5–10 ms          |
| CK5 canonicalization  | 30–80 ms (warm)  |

Total save-path overhead ≈ 50–130 ms warm. Well under perceived latency
for a "Save" click. Autosave skips canonicalization (see §9).

⚠ uncertain — these are projections; benchmark on the real golden before
quoting them to stakeholders.

---

## 12. Test strategy

- **Unit tests per allow/deny rule.** Table-driven, one row per rule,
  known-bad input → expected `{ rule }` in errors. Known-good input → no
  errors.
- **XSS corpus.** Maintain `fixtures/xss/*.html` with real-world payloads
  (OWASP cheatsheet, CK5 advisory PoCs, SVG exploits). Every file must
  round-trip through sanitize + validate without surfacing an active
  handler.
- **Snapshot tests for canonicalized output.** A small set of
  representative documents; snapshots committed. Changes require a PR
  note ("canonicalizer output changed due to CK5 vX.Y bump").
- **Property tests (optional).** Fuzz with random-but-allowlist-valid
  HTML; assert sanitizer is a no-op on its own output (idempotence).
- **Integration test.** End-to-end save → fetch → render in a fresh
  editor; assert no warnings in CK5's schema filter logs.

---

## Open questions

- ⚠ uncertain — whether `style` attributes are needed anywhere in our
  editor UX (§5). Default: banned. Revisit if a print feature requires it.
- ⚠ uncertain — sync vs async canonicalization (§9).
- ⚠ uncertain — whether `data:` image support is worth the audit burden,
  or whether we require CDN upload for every image.
- ⚠ uncertain — `sanitize-html` CSS parser completeness if we do allow
  `style`. May need to pivot to DOMPurify for stricter enforcement.
- ⚠ uncertain — how the `path` selector in error reports maps back to
  CK5 model positions reliably across paste/undo. Prototype in the
  editor integration pass.

---

## Sources & cross-refs

- [04 — Schema](./04-schema.md)
- [07 — Markers](./07-markers.md)
- [10 — Restricted editing](./10-restricted-editing.md)
- [18 — HTML support (GHS)](./18-html-support.md)
- [24 — Data format](./24-data-format.md)
- [25 — MDDM IR bridge](./25-mddm-ir-bridge.md)
- [26 — Autosave & persistence](./26-autosave-persistence.md)
- [35 — Backend contracts](./35-backend-contracts.md)
- [36 — Server rendering](./36-server-rendering.md)

External:

- https://www.npmjs.com/package/sanitize-html
- https://github.com/cure53/DOMPurify
- https://www.npmjs.com/package/linkedom
- https://owasp.org/www-community/xss-filter-evasion-cheatsheet
