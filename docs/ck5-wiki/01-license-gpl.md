---
title: License — GPL
status: draft
area: foundation
---

# 01 — License (GPL-2.0-or-later)

MetalDocs embeds CKEditor 5 under its open-source tier. This page documents
what the GPL license covers, how to configure it, and what we must avoid or
reimplement because it lives behind the commercial tier.

## Dual-license model

CKEditor 5 ships under a dual license:

- **GPL-2.0-or-later** — free for any project that itself complies with GPL.
  All "open-source" plugins in the main `@ckeditor/ckeditor5-*` packages are
  available under this license.
- **Commercial license** — required when (a) the consuming product is not
  GPL-compatible, or (b) the integration loads any *premium* plugin
  (Collaboration, Pagination, Export, Import, AI Assistant, CKBox, etc.).

There is no middle tier. Either you accept GPL obligations, or you buy a
commercial key. Choosing GPL also forbids loading premium bundles — the
license key gate rejects them at runtime.

## `licenseKey` configuration

Since CKEditor 5 v38+ (and strictly enforced on v44+), a `licenseKey` is
required even for the GPL tier.

```ts
ClassicEditor.create(element, {
  licenseKey: 'GPL',
  // ...plugins, toolbar, etc.
});
```

Behavior:

- `licenseKey: 'GPL'` — unlocks every plugin shipped under the GPL tier.
  No network call, no telemetry, no expiration.
- Missing / omitted `licenseKey` — recent versions print a console warning
  and, on newer builds, refuse to initialize. Treat it as mandatory.
- Commercial key (JWT from the CKEditor customer portal) — required to load
  any premium plugin. Loading a premium plugin with `'GPL'` throws a
  `license-key-invalid-license-key` or plugin-specific error at editor
  bootstrap.

The key is read once during `Editor.create()`; it cannot be swapped at
runtime.

## Included under GPL

The following are GPL-tier and fully usable in MetalDocs. Plugin package
names given where relevant.

**Core / framework**

- `@ckeditor/ckeditor5-core` — editor, plugin system, commands
- `@ckeditor/ckeditor5-engine` — model, view, schema, conversion, differ
- `@ckeditor/ckeditor5-ui` — toolbar, balloons, dropdowns
- `@ckeditor/ckeditor5-utils`, `-widget`, `-typing`, `-enter`, `-select-all`
- `@ckeditor/ckeditor5-undo` — undo/redo
- `@ckeditor/ckeditor5-clipboard` — copy/paste, drag & drop
- `@ckeditor/ckeditor5-watchdog` — crash recovery

**Editing features**

- `-heading`, `-paragraph`, `-basic-styles` (bold/italic/underline/strike/code/sub/sup)
- `-font` (family, size, color, background color)
- `-alignment`
- `-list` (numbered, bulleted, to-do list, list properties)
- `-indent` (indent/outdent, block indent)
- `-block-quote`, `-horizontal-line`, `-page-break`, `-special-characters`
- `-code-block`
- `-link` (with `linkimage`, custom link decorators), `-autolink`
- `-image` (`imageblock`, `imageinline`, `imagecaption`, `imagestyle`,
  `imagetoolbar`, `imageresize`, `imageupload` adapter interface)
- `-table` (table, table toolbar, table properties, table cell properties,
  table caption, column resize)
- `-media-embed`
- `-mention` (base feed mechanics — premium UI layer is separate; see below)
- `-restricted-editing` (standard + exception; span-only exceptions in
  practice — see MetalDocs memory)
- `-find-and-replace`
- `-remove-format`, `-source-editing` (plain HTML source view)
- `-word-count`
- `-html-support` (General HTML Support / GHS)
- `-html-embed` (raw HTML block)
- `-style` (named style dropdown)
- `-show-blocks`
- `-bookmark` ⚠ uncertain — verify (ships in recent versions; believed GPL)
- `-autoformat`, `-autosave`
- `-markdown-gfm` (Markdown data processor)
- `-easy-image` client *interface* is GPL, but Easy Image **service** is a
  paid CKEditor Cloud Services product — do not confuse the two.

## NOT included (premium / commercial only)

These require a paid license key and are explicitly unavailable to
MetalDocs under the GPL choice.

- **Real-Time Collaboration** — presence, selection sync, remote cursors
  (`@ckeditor/ckeditor5-real-time-collaboration`)
- **Comments** (`-comments`) — standalone comments feature; also dependency
  for TC/RH
- **Track Changes** (`-track-changes`)
- **Revision History** (`-revision-history`)
- **Pagination** (`-pagination`) — page-break visualization, print-accurate
  page boxes. This is the feature MetalDocs must reimplement for MDDM.
- **Export to Word** (`-export-word`) — server-side DOCX conversion
- **Export to PDF** (`-export-pdf`) — server-side PDF conversion
- **Import from Word** (`-import-word`) — server-side DOCX → CKEditor data
- **Paste from Office Enhanced** (`-paste-from-office-enhanced`) — note:
  the basic `-paste-from-office` *is* GPL; the "enhanced" variant with
  high-fidelity Word/Excel ingestion is premium.
- **Case Change** (`-case-change`)
- **CKBox** — file/image manager, cloud service
- **AI Assistant** (`-ai`)
- **Multi-level List** (`-multi-level-list`) — legal-style numbered lists;
  premium as of v40+. Note: standard `-list` + list properties remain GPL.
- **Mentions — premium UI** ⚠ uncertain — verify. Core `-mention` plugin
  is GPL; some marketplace/enhanced mention features may be premium.
- **Email editing / Email configuration / Email integration** — the
  dedicated Email suite added in recent versions is premium.
- **Merge Fields** (`-merge-fields`) — template placeholder feature
- **Slash commands** (`-slash-command`) ⚠ uncertain — verify; believed
  premium in current releases.
- **Productivity Pack** — bundle marketing name covering Slash commands,
  Format Painter, Table of Contents, Templates, Document Outline, Case
  Change. All members premium.
- **Templates** (`-template`) — premium template insertion feature
- **Table of Contents** (`-document-outline`, `-table-of-contents`)
- **Format Painter** (`-format-painter`)
- **Content Insights / Asset Manager** (Cloud Services products)

Anything not in the "Included" list above should be assumed premium until
verified against the current CKEditor pricing/packages page.

## GPL obligations

GPL-2.0-or-later is copyleft. Practical obligations when we ship MetalDocs
as a product that embeds CKEditor 5:

1. **License notice**: preserve CKEditor 5 copyright headers and include
   the GPL-2.0 license text with any distribution (source or binary). Our
   `NOTICES` / `THIRD_PARTY_LICENSES` file must list it.
2. **Source availability**: when we *distribute* the software (not just
   SaaS-host it), we must offer corresponding source to recipients. For a
   purely hosted web app this obligation is less strict — GPL-2.0 does not
   have the AGPL network-use clause — but we still distribute the JS
   bundle to browsers. The conservative reading: expose a link to our
   CKEditor 5 source (the unmodified upstream plus our build config).
3. **Downstream license compatibility**: any code we *link with* CKEditor
   5 (i.e. the editor configuration module and its direct dependents) is
   arguably a derivative work. Keep MetalDocs editor-adjacent code under
   GPL-compatible terms, or maintain a clean architectural boundary so
   the rest of the product can be licensed separately.
4. **No additional restrictions**: we cannot add DRM, anti-reverse-
   engineering clauses, or field-of-use limits on the CKEditor portion.
5. **Modifications**: if we patch CKEditor 5 itself (fork), the patched
   source must be made available to recipients of the modified binary.

We do not need to publish the *entire* MetalDocs codebase — only the
CKEditor 5 portion and anything that counts as a derivative work of it.
Exact scope of "derivative work" for linked JS is legally grey; treat as a
project-level legal question, not a technical one.

## Practical implications for MetalDocs

- **No premium plugins at build time.** The bundler config, plugin list,
  and editor factory (`apps/ck5-studio/src/lib/editorConfig.ts`) must
  reference only GPL-tier packages.
- **`licenseKey: 'GPL'` is the canonical config value.** It should be set
  in one place and imported by all editor entry points.
- **Pagination must be built in-house.** MDDM pagination (see
  `docs/superpowers/plans/2026-04-14-mddm-professional-pagination-remediation.md`)
  is a first-class project deliverable precisely because `-pagination` is
  off-limits.
- **Export/import pipelines are ours.** DOCX export goes through
  HTMLtoDOCX; DOCX import (if needed) must be a custom data processor.
  No `-export-word` / `-import-word`.
- **Comments / Track Changes / Revision History are not options.** If
  review workflows are needed, build on top of `-restricted-editing` +
  application-level state; do not emulate CKEditor's premium comment API
  1:1.
- **Case Change, Format Painter, TOC, Templates, Slash command** — if any
  of these appear in product requirements, flag them as premium and
  propose a GPL-tier substitute before scoping work.
- **CKBox replacement.** Image/asset upload must be implemented against
  the GPL `imageupload` adapter interface, pointing at our own storage.
- **Paste from Office.** Basic `-paste-from-office` is enough for most
  Word paste cases. The "enhanced" fidelity is premium; accept the
  resulting downgrade and document known paste limitations instead.
- **Legal / marketing copy.** Product pages should reference "CKEditor 5
  (GPL-2.0-or-later)" and link to the upstream repo, not imply we are a
  commercial CKEditor customer.

## Open questions

- Exact list of premium-only plugins to avoid on v48.
- Does `licenseKey: 'GPL'` unlock everything GPL-tier on v48?
- GPL obligations for our own distribution (source offer, notices).
- Is `-bookmark` shipped under GPL in v48, or bundled with a premium pack?
- Current status of `-mention` sub-features — which pieces moved to
  premium in v44–v48?
- Is `-slash-command` GPL or Productivity Pack on v48?
- For a browser-delivered SPA, what is legal's interpretation of "source
  availability" under GPL-2.0 for the CKEditor bundle we ship?
- Does restricted-editing interact with the license gate in any way
  (no known issue, but worth confirming before relying on it in locked
  templates)?
- Version pin: which CKEditor 5 minor is MetalDocs targeting long-term,
  and does that version's premium/free split match this document?

## Sources

Internal knowledge only — no external links supplied.
