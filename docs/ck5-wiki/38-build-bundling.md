---
title: Build & bundling
status: draft
area: ops
priority: HIGH
---

# 38 — Build & bundling

How CK5 v48 is installed, imported, and bundled in a Vite + React app. Scope: the
single-package model (`ckeditor5`), CSS pipeline, tree-shaking reality, and Vite /
SSR gotchas. Proven in this repo under `apps/ck5-studio` (Vite + React 18, CK5
v48, DecoupledEditor).

## 1. v48 single-package install

Since v42 (hardened in v44–v48), CK5 ships as **one** npm package. The legacy
`@ckeditor/ckeditor5-<feature>` packages are gone — do not install them, they are
not published for v48.

```bash
npm install ckeditor5 @ckeditor/ckeditor5-react
```

- `ckeditor5` — core + all first-party plugins (editor classes, features, UI).
- `@ckeditor/ckeditor5-react` — the `<CKEditor>` React wrapper component.
- Premium features live in `ckeditor5-premium-features` (separate install, not
  used in MetalDocs).

There is no "build" step to run on the package. You import the classes you want
and pass them to `ClassicEditor.create()` / `DecoupledEditor.create()` at runtime.

## 2. Plugin import pattern

All plugins, editor classes, and UI helpers are **named exports of `ckeditor5`**.

```ts
import {
  DecoupledEditor,
  Essentials, Paragraph, Heading, Bold, Italic, Underline,
  Link, List, Table, TableToolbar,
  Image, ImageUpload, ImageToolbar, Base64UploadAdapter,
  Autoformat, Alignment, FontFamily, FontSize, FontColor,
  RestrictedEditingMode, StandardEditingMode,
} from "ckeditor5";
```

This repo uses a `import * as CKEditor from "ckeditor5"` + destructure pattern
(see `apps/ck5-studio/src/lib/editorConfig.ts`) to sidestep a TS typing issue
where some feature classes are typed loosely; this works but loses per-name type
inference. ⚠ uncertain whether that workaround is still needed in v48.4+.

Deep subpath imports like `ckeditor5/src/core` are **not** part of the public API
and should not be used.

## 3. CSS imports

One stylesheet covers the entire editor UI and content:

```ts
import "ckeditor5/ckeditor5.css";
```

Import it **once** at the application entry (in this repo: `src/main.tsx`).

### Order with Tailwind / tokens

Tailwind's `@tailwind base` applies a global reset that will stomp CK5 UI
spacing if loaded after. Recommended order:

```ts
import "ckeditor5/ckeditor5.css";   // 1. editor chrome + content
import "./styles/tokens.css";       // 2. design tokens
import "./styles/base.css";         // 3. your reset / Tailwind base
import "./styles/app.css";          // 4. app overrides (can target .ck-*)
```

Rationale: CK5 ships its own low-specificity styles scoped under `.ck` and
`.ck-content`. Loading app CSS **after** CK5's lets you override cleanly without
`!important`. If Tailwind's `preflight` resets reach CK5 elements, scope
Tailwind with `corePlugins.preflight: false` in content areas, or wrap the
editor in an element where preflight is neutralised.

There is no separate "content-styles.css" export in v48 — `ckeditor5.css`
contains both UI and content styles. ⚠ uncertain whether a content-only bundle
is published; we import the combined file.

## 4. Vite config

**None required.** No plugins, no aliases, no `optimizeDeps` hack. The
`apps/ck5-studio/vite.config.ts` in this repo does not mention CKEditor and
builds/HMRs cleanly.

Things you do **not** need in v48 (unlike v41 and earlier):
- No `@ckeditor/ckeditor5-dev-utils` / `styles.getPostCssConfig()`.
- No `raw-loader` / `?raw` imports for SVGs — icons are inlined in the package.
- No PostCSS plugin for CK5.
- No worker or WASM config. CK5 runs on the main thread and ships no wasm.

If you use `vite-plugin-dts` or strict `build.target: esnext`, no adjustments
needed.

## 5. Tree-shaking: partial, not free

Importing named plugins from `ckeditor5` **does** let Rollup drop feature
classes you never reference, but in practice the win is modest:

- Core modules (engine, UI framework, utils, typing, widget, enter, clipboard)
  are pulled in by almost every plugin, so ~70% of the bundle is unavoidable
  once you include `Essentials + Paragraph`.
- Icons and translations are in separate chunks and are dropped if unused.
- A minimal classic editor still lands around **400–600 KB** gzipped. ⚠
  uncertain on the exact figure for v48; measure with `vite build --report`.

Tactics that actually reduce weight:
- Lazy-load the editor module (see §8) so it is not in the initial route chunk.
- Do not import the whole `ckeditor5` namespace if you need tree-shaking — use
  named imports. (The `import *` pattern in `editorConfig.ts` is fine because
  Rollup can still shake named properties in production mode, but named
  imports are safer.) ⚠ uncertain.

## 6. Translations / locale loading

UI translations live under `ckeditor5/translations/<lang>.js` and are imported
side-effect-style before `.create()`:

```ts
import "ckeditor5/translations/pt.js";
// or
import coreTranslations from "ckeditor5/translations/pt.js";

await DecoupledEditor.create(el, {
  language: "pt",
  translations: [coreTranslations], // only needed if you pass explicitly
  plugins: [...],
});
```

English (`en`) is the default and requires no import. For premium features
(`ckeditor5-premium-features/translations/<lang>.js`) you would import and
merge into the `translations` array. MetalDocs currently ships en-US only and
does not import any translation file.

⚠ uncertain on whether `.js` or bare path (`ckeditor5/translations/pt`) is
canonical in v48 — the repo has no translation import to verify against.

## 7. TypeScript types

- Types ship **inside** the `ckeditor5` package (`types/` folder referenced from
  `package.json#exports`). No `@types/ckeditor__ckeditor5-*` needed — those
  packages are obsolete and should not be installed.
- `@ckeditor/ckeditor5-react` ships its own `.d.ts`.
- Known gotcha: some plugin classes are exported as `any`-ish/loose types when
  consumed through `import *`. Destructuring through
  `CKEditor as Record<string, any>` (this repo's pattern) suppresses those
  false positives but sacrifices autocomplete. Prefer direct named imports for
  classes you need strongly typed.
- `DecoupledEditor.create()` returns `Promise<DecoupledEditor>`; `ui.view.toolbar`
  can be `null` before ready, so guard before mounting the detached toolbar.

## 8. SSR safety

**CK5 is a browser-only library.** It touches `window`, `document`, `navigator`,
and DOM ranges at import time in a few submodules. It will crash under Node
SSR.

Patterns:

- **Vite SPA (this repo):** no SSR — import normally. ✅
- **Next.js App Router:** mark the editor component `"use client"` AND
  dynamic-import it with `{ ssr: false }`:

  ```tsx
  const Editor = dynamic(() => import("./Editor"), { ssr: false });
  ```

- **Vite + vite-ssr / Remix / Astro islands:** wrap the editor mount in a
  `useEffect` (never in the render body), and gate module import with
  `if (typeof window !== "undefined")` dynamic `import()`.
- Never import `ckeditor5` at the top level of a file that is evaluated on the
  server.

## 9. Vite gotchas

Collected from this repo and community reports:

- **Dev-server cold start is slow the first time.** Vite pre-bundles `ckeditor5`
  into `node_modules/.vite/deps/` — expect a 5–15 s pause on first `vite dev`
  after install or when `ckeditor5` changes version. Subsequent starts are
  cached. Don't "fix" this with manual `optimizeDeps.exclude` unless you see a
  real bug; excluding it hurts dev performance.
- **HMR and editor lifecycle.** The `<CKEditor>` React wrapper handles
  destroy-on-unmount correctly, but a manual `DecoupledEditor.create()` +
  `useEffect` flow (as used here) **must** call `editor.destroy()` in the
  cleanup function. On HMR, a missing cleanup leaves multiple editor instances
  attached to the same DOM node — visible as duplicated toolbars or
  "Cannot create two editors on one element" errors.
- **StrictMode double-mount (React 18).** The effect that creates the editor
  runs twice in dev. The cleanup-then-create pattern must handle that; guard
  with a cancelled/disposed flag or await destroy before re-creating.
- **Production build** (`vite build`) produces a large single vendor chunk by
  default. If you care about initial-route weight, put the editor behind a
  route-level `React.lazy()` / dynamic `import()`.
- **Source maps.** `ckeditor5` ships source maps; Vite serves them in dev and
  includes them in build by default. Disable via `build.sourcemap: false` if
  your deployment does not want them.
- **Icon SVGs** are inlined into JS — no asset-loader configuration needed, but
  they inflate chunk size slightly. This is expected.

## 10. Minimal working example (Vite + React + CK5 v48)

```tsx
// src/main.tsx
import { StrictMode } from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "ckeditor5/ckeditor5.css";
import "./styles/app.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <StrictMode><App /></StrictMode>,
);
```

```tsx
// src/Editor.tsx
import { useEffect, useRef } from "react";
import {
  ClassicEditor, Essentials, Paragraph, Heading, Bold, Italic, Link, List,
} from "ckeditor5";

export function Editor() {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    let editor: ClassicEditor | null = null;
    let disposed = false;

    ClassicEditor.create(ref.current!, {
      licenseKey: "GPL",
      plugins: [Essentials, Paragraph, Heading, Bold, Italic, Link, List],
      toolbar: ["heading", "|", "bold", "italic", "link", "bulletedList", "numberedList"],
    }).then((instance) => {
      if (disposed) { instance.destroy(); return; }
      editor = instance;
    });

    return () => {
      disposed = true;
      editor?.destroy();
    };
  }, []);

  return <div ref={ref} />;
}
```

```ts
// vite.config.ts — nothing CK5-specific needed
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
export default defineConfig({ plugins: [react()] });
```

```json
// package.json (excerpt)
{
  "dependencies": {
    "ckeditor5": "^48.0.0",
    "@ckeditor/ckeditor5-react": "^9.0.0",
    "react": "^18.3.0",
    "react-dom": "^18.3.0"
  }
}
```

For DecoupledEditor (used in `apps/ck5-studio`), swap `ClassicEditor` for
`DecoupledEditor` and mount the toolbar via `editor.ui.view.toolbar.element`.

## Repo references

- `apps/ck5-studio/src/main.tsx` — CSS import order.
- `apps/ck5-studio/src/lib/editorConfig.ts` — plugin import pattern,
  DecoupledEditor, restricted-editing wiring.
- `apps/ck5-studio/vite.config.ts` — no CK5-specific Vite config (proof that
  none is needed).

## Open questions

- Exact gzipped bundle delta for `Essentials + Paragraph + Bold` vs. full
  plugin set in v48. Needs a `vite build --report` measurement.
- Whether `import * as CKEditor from "ckeditor5"` truly tree-shakes in Rollup
  production mode, or whether named imports are measurably smaller.
- Canonical translations import path for v48 (`ckeditor5/translations/pt.js`
  vs. `ckeditor5/translations/pt`) — not exercised in repo yet.
- Is there a separate content-only CSS export, or is `ckeditor5.css` the only
  stylesheet for both UI and content?
- Does Tailwind `preflight` need explicit scoping around the editor, or do
  CK5's selectors already win on specificity?
- React 19 / `@ckeditor/ckeditor5-react` v9 compatibility matrix (repo is on
  React 18).

## Sources

- https://ckeditor.com/docs/ckeditor5/latest/framework/index.html
- https://ckeditor.com/docs/ckeditor5/latest/getting-started/index.html
- https://ckeditor.com/docs/ckeditor5/latest/getting-started/quick-start.html
- Repo: `apps/ck5-studio/src/main.tsx`, `src/lib/editorConfig.ts`, `vite.config.ts`
- CKEditor 5 changelog v42–v48 (single-package migration notes)
