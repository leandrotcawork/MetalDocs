---
title: Versioning & v48 distribution
status: draft
area: foundation
---

# 02 — Versioning & v48 distribution

Single-package `ckeditor5` install model, removal of custom builds, semver.

## Current versions (as of 2026-04-15)

- `ckeditor5` (npm): latest stable **v48.0.0** (released 2025-03-31), with a follow-up **v47.6.2** patch on the v47 line (2025-04-08). ⚠ uncertain — verified via GitHub Releases; direct npm pages returned 403 to our fetcher, so newer patches on the v48 line may exist.
- `@ckeditor/ckeditor5-react` (npm): ⚠ uncertain — the npm page was unreachable (403). Project memory pins it at **`@ckeditor/ckeditor5-react@11.1.1`**; the React wrapper is expected to track CK5 majors, but we could not confirm a newer tag.
- Release cadence: historically roughly **monthly** minor/patch releases, with majors a few times per year. ⚠ uncertain on exact cadence for 2025–2026.

## The v42+ "new installation methods" (NIM)

Starting with **v42.0.0** (mid-2024), CKEditor 5 introduced a new distribution model. In **v48.0.0** this became **mandatory** — the old installation paths were removed.

### What went away

- Predefined builds (`@ckeditor/ckeditor5-build-classic`, `-inline`, `-balloon`, etc.).
- The Online Builder / customized builds workflow.
- DLL builds.
- Per-feature npm packages as the primary install surface: packages no longer ship `src/`, `theme/`, and `lang/` directories to npm.
- Mandatory CKEditor-specific webpack/postcss/svg loader configuration.

### What replaced it

- A single umbrella package, **`ckeditor5`**, that re-exports every open-source plugin and the editor classes.
- A parallel **`ckeditor5-premium-features`** package for commercial features (track changes, comments, real-time collaboration, export-to-PDF/Word, etc.).
- Plain CSS files imported separately from JS — no bundler plugins required.
- Translations as plain JS objects passed via `config.translations`, not side-effect imports.
- CDN distribution as a first-class alternative for no-build setups.

## Import pattern under the new model

### JS imports

```ts
import {
  ClassicEditor,
  Essentials,
  Bold,
  Italic,
  Paragraph,
} from 'ckeditor5';
import { FormatPainter, SlashCommand } from 'ckeditor5-premium-features';
```

Everything editor-related is pulled from the `ckeditor5` (and optionally `ckeditor5-premium-features`) entry points. Confirmed from the official NIM migration guide.

### CSS imports

```ts
import 'ckeditor5/ckeditor5.css';
import 'ckeditor5-premium-features/ckeditor5-premium-features.css';
```

A single CSS file per package. ⚠ uncertain — finer-grained CSS splits (e.g. per-feature stylesheets) may exist for optimization but are not documented in the pages we could reach.

### Minimal usage (vanilla)

```ts
import { ClassicEditor, Essentials, Bold, Italic, Paragraph } from 'ckeditor5';
import 'ckeditor5/ckeditor5.css';

ClassicEditor.create(document.querySelector('#editor')!, {
  plugins: [Essentials, Paragraph, Bold, Italic],
  toolbar: ['bold', 'italic'],
  licenseKey: '<YOUR_LICENSE_KEY_OR_GPL>',
});
```

Note: from v44 onward a `licenseKey` is required even for GPL usage (`'GPL'` literal). ⚠ uncertain on exact version where this became mandatory — confirm against release notes.

## React wrapper usage

```tsx
import { CKEditor } from '@ckeditor/ckeditor5-react';
import { ClassicEditor, Essentials, Bold, Italic, Paragraph } from 'ckeditor5';
import 'ckeditor5/ckeditor5.css';

<CKEditor
  editor={ClassicEditor}
  data="<p>Hello</p>"
  config={{
    plugins: [Essentials, Paragraph, Bold, Italic],
    toolbar: ['bold', 'italic'],
    licenseKey: 'GPL',
  }}
  disabled={false}
  onReady={(editor) => { /* grab instance */ }}
  onChange={(event, editor) => { /* editor.getData() */ }}
  onBlur={(event, editor) => {}}
  onFocus={(event, editor) => {}}
  onError={(error, { phase, willEditorRestart }) => {}}
/>
```

Props set confirmed against the general shape documented for the React wrapper. ⚠ uncertain — we could not open the current React integration docs page directly; validate prop names and signatures against the installed `@ckeditor/ckeditor5-react` version's types.

## Semver & release cadence

- CKEditor 5 follows **semver**: majors (`X.0.0`) may remove APIs and installation methods, minors add features, patches fix bugs.
- Majors have been issued several times per year; v48 landed in March 2025.
- `ckeditor5` and `ckeditor5-premium-features` are **version-locked** — always install the same version of both. ⚠ uncertain phrasing but consistent with NIM guidance.
- The React/Vue/Angular wrappers version independently but must be paired with a compatible `ckeditor5` major.

## Migration notes (from v41 or earlier)

If a codebase predates NIM:

1. Remove all `@ckeditor/ckeditor5-*` feature packages from `package.json` except the React/Vue/Angular wrapper.
2. Add `ckeditor5` (and `ckeditor5-premium-features` if licensed) at the same version.
3. Rewrite imports to pull from `'ckeditor5'` / `'ckeditor5-premium-features'`.
4. Add the CSS imports (`ckeditor5/ckeditor5.css`, premium equivalent).
5. Drop CKEditor-specific webpack/rollup config (postcss loader, svg loader, translations plugin).
6. Convert translation side-effect imports to JS-object imports passed via `config.translations`.
7. Add `licenseKey` (use `'GPL'` if you're on the GPL license).
8. Re-check any custom plugins: they must now import their base classes from `'ckeditor5'` instead of the old per-package paths.

Predefined builds (`ckeditor5-build-classic` etc.) have **no drop-in replacement** — you assemble an equivalent plugin list yourself from `ckeditor5`.

## Known breaking changes in recent majors

**v48.0.0** (2025-03-31):
- Old installation methods fully removed; NIM is mandatory.
- Table alignment now emitted as CSS classes instead of inline styles by default.
- Export-to-PDF defaults to API v2 (output may differ).
- Root-related config consolidated under `config.root` / `config.roots`.
- DLL builds removed.
- Packages no longer ship `src/`, `theme/`, `lang/` to npm.

**v47.6.0** (2025-03-04):
- Security fix: XSS in General HTML Support when using unsafe markup configurations. Upgrade if you enable GHS. ⚠ uncertain on full advisory scope.

**v42.0.0** (mid-2024):
- Introduced NIM (then optional).
- `licenseKey` became more prominent in config. ⚠ uncertain on exact version gate.

## Open questions

- Exact current published version of `ckeditor5` and `@ckeditor/ckeditor5-react` on npm (npm fetches returned 403 — verify with `npm view ckeditor5 version` locally).
- Whether v48.x has shipped patch releases (48.0.1+) between 2025-03-31 and today.
- Canonical CSS split paths beyond `ckeditor5/ckeditor5.css` (e.g. editor-only vs content-only styles).
- Precise React wrapper major paired with `ckeditor5@48` — our pinned `11.1.1` predates v48 and may need bumping.
- Exact version where `licenseKey` became mandatory for GPL use.
- Release cadence commitment for 2026 (still monthly?).

## Sources

- https://github.com/ckeditor/ckeditor5/releases
- https://ckeditor.com/docs/ckeditor5/latest/updating/index.html
- https://ckeditor.com/docs/ckeditor5/latest/updating/nim-migration/migration-to-new-installation-methods.html
- https://ckeditor.com/docs/ckeditor5/latest/getting-started/installation/quick-start.html
- https://www.npmjs.com/package/ckeditor5 (attempted — 403)
- https://www.npmjs.com/package/@ckeditor/ckeditor5-react (attempted — 403)
