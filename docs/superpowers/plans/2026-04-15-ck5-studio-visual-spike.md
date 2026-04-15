# CKEditor 5 Studio Visual Spike Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build disposable standalone CKEditor 5 visual spike at `apps/ck5-studio/` to evaluate shell styling, native toolbar quality, and template-like block insertion without touching MetalDocs backend or production frontend.

**Architecture:** Create isolated React + Vite app that owns its own dependencies, styles, and browser tests. Use CKEditor 5 `DecoupledEditor` with native toolbar mounted into shell chrome, plus a left palette that inserts starter snippets and a right panel that reflects selection summary and visual-only controls. Keep all state local, with optional `localStorage` restore for convenience, and verify result in real Chromium with screenshots.

**Tech Stack:** React 18, TypeScript, Vite, CKEditor 5 decoupled editor, `@ckeditor/ckeditor5-react`, Vitest, Playwright, CSS.

---

## File Structure

**Create**
- `apps/ck5-studio/package.json`
- `apps/ck5-studio/tsconfig.json`
- `apps/ck5-studio/vite.config.ts`
- `apps/ck5-studio/playwright.config.ts`
- `apps/ck5-studio/index.html`
- `apps/ck5-studio/src/main.tsx`
- `apps/ck5-studio/src/App.tsx`
- `apps/ck5-studio/src/types.ts`
- `apps/ck5-studio/src/components/AppShell.tsx`
- `apps/ck5-studio/src/components/TopBar.tsx`
- `apps/ck5-studio/src/components/LeftLibrary.tsx`
- `apps/ck5-studio/src/components/EditorCanvas.tsx`
- `apps/ck5-studio/src/components/RightPanel.tsx`
- `apps/ck5-studio/src/lib/editorConfig.ts`
- `apps/ck5-studio/src/lib/contentSnippets.ts`
- `apps/ck5-studio/src/lib/localDraft.ts`
- `apps/ck5-studio/src/lib/selectionSummary.ts`
- `apps/ck5-studio/src/styles/tokens.css`
- `apps/ck5-studio/src/styles/base.css`
- `apps/ck5-studio/src/styles/app.css`
- `apps/ck5-studio/src/__tests__/contentSnippets.test.ts`
- `apps/ck5-studio/src/__tests__/localDraft.test.ts`
- `apps/ck5-studio/src/__tests__/selectionSummary.test.ts`
- `apps/ck5-studio/tests/editorial-shell.spec.ts`

**Modify**
- `tasks/lessons.md`

**Responsibilities**
- `package.json`: isolated scripts and deps for throwaway spike
- `editorConfig.ts`: CKEditor plugin/config assembly
- `contentSnippets.ts`: palette insert HTML/snippet sources
- `localDraft.ts`: save/load local-only draft
- `selectionSummary.ts`: map current selection/element to right-panel summary
- `EditorCanvas.tsx`: CKEditor mount, toolbar host, insert APIs, image path
- `AppShell.tsx`: compose top bar, left library, center editor, right panel
- `app.css` + token/base files: editorial reference styling
- `editorial-shell.spec.ts`: real Chromium smoke + screenshot capture

---

### Task 1: Scaffold Isolated CK5 Studio App

**Files:**
- Create: `apps/ck5-studio/package.json`
- Create: `apps/ck5-studio/tsconfig.json`
- Create: `apps/ck5-studio/vite.config.ts`
- Create: `apps/ck5-studio/index.html`
- Create: `apps/ck5-studio/src/main.tsx`
- Create: `apps/ck5-studio/src/App.tsx`

- [ ] **Step 1: Write failing build target by creating package manifest and empty app entry**

```json
{
  "name": "@metaldocs/ck5-studio",
  "private": true,
  "version": "0.0.1",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc --noEmit && vite build",
    "preview": "vite preview",
    "test": "vitest run",
    "test:watch": "vitest",
    "e2e": "playwright test"
  },
  "dependencies": {
    "@ckeditor/ckeditor5-react": "^9.2.0",
    "@fontsource/inter": "^5.2.29",
    "@fontsource/newsreader": "^5.2.8",
    "ckeditor5": "^46.0.0",
    "react": "^18.3.1",
    "react-dom": "^18.3.1"
  },
  "devDependencies": {
    "@playwright/test": "^1.58.2",
    "@types/react": "^18.3.12",
    "@types/react-dom": "^18.3.1",
    "@vitejs/plugin-react": "^4.3.1",
    "jsdom": "^25.0.1",
    "typescript": "^5.6.3",
    "vite": "^5.4.10",
    "vitest": "^2.1.9"
  }
}
```

```ts
// apps/ck5-studio/src/main.tsx
import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
```

```tsx
// apps/ck5-studio/src/App.tsx
export default function App() {
  return <div>CK5 Studio</div>;
}
```

- [ ] **Step 2: Run build to verify it fails before full config exists**

Run: `npm.cmd --prefix apps/ck5-studio run build`

Expected: FAIL with missing `tsconfig.json` or Vite config.

- [ ] **Step 3: Add minimal TypeScript + Vite config**

```json
// apps/ck5-studio/tsconfig.json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "Bundler",
    "allowImportingTsExtensions": false,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "types": ["vite/client"]
  },
  "include": ["src"]
}
```

```ts
// apps/ck5-studio/vite.config.ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    host: "127.0.0.1",
    port: 4175,
  },
  preview: {
    host: "127.0.0.1",
    port: 4175,
  },
});
```

```html
<!-- apps/ck5-studio/index.html -->
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>CK5 Studio Spike</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
  </html>
```

- [ ] **Step 4: Run build to verify scaffold passes**

Run: `npm.cmd --prefix apps/ck5-studio run build`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/ck5-studio/package.json apps/ck5-studio/tsconfig.json apps/ck5-studio/vite.config.ts apps/ck5-studio/index.html apps/ck5-studio/src/main.tsx apps/ck5-studio/src/App.tsx
git commit -m "feat(ck5-studio): scaffold standalone spike app"
```

---

### Task 2: Mount Decoupled CKEditor With Native Toolbar

**Files:**
- Create: `apps/ck5-studio/src/lib/editorConfig.ts`
- Create: `apps/ck5-studio/src/components/EditorCanvas.tsx`
- Modify: `apps/ck5-studio/src/App.tsx`

- [ ] **Step 1: Write failing editor config through missing imports**

```ts
// apps/ck5-studio/src/lib/editorConfig.ts
import {
  Alignment,
  AutoImage,
  Autoformat,
  Base64UploadAdapter,
  BlockQuote,
  Bold,
  DecoupledEditor,
  Essentials,
  FontBackgroundColor,
  FontColor,
  FontFamily,
  FontSize,
  Heading,
  Image,
  ImageCaption,
  ImageInsert,
  ImageResize,
  ImageStyle,
  ImageToolbar,
  ImageUpload,
  Italic,
  Link,
  List,
  Paragraph,
  Table,
  TableToolbar,
  Underline,
} from "ckeditor5";

export const editorClass = DecoupledEditor;

export const editorConfig = {
  licenseKey: "GPL",
  plugins: [
    Essentials,
    Paragraph,
    Heading,
    Bold,
    Italic,
    Underline,
    Link,
    List,
    Table,
    TableToolbar,
    Image,
    ImageUpload,
    ImageToolbar,
    ImageStyle,
    ImageResize,
    ImageCaption,
    ImageInsert,
    AutoImage,
    Base64UploadAdapter,
    Autoformat,
    Alignment,
    FontFamily,
    FontSize,
    FontColor,
    FontBackgroundColor,
    BlockQuote,
  ],
  toolbar: {
    items: [
      "undo", "redo", "|", "heading", "|",
      "fontFamily", "fontSize", "fontColor", "fontBackgroundColor", "|",
      "bold", "italic", "underline", "link", "|",
      "alignment", "bulletedList", "numberedList", "blockQuote", "insertTable", "uploadImage"
    ],
    shouldNotGroupWhenFull: true
  }
};
```

- [ ] **Step 2: Run dev server to verify package/import issues surface early**

Run: `npm.cmd --prefix apps/ck5-studio run dev`

Expected: first failure if CKEditor packages not yet installed locally. Install deps, rerun until app starts.

- [ ] **Step 3: Add decoupled editor mount component**

```tsx
// apps/ck5-studio/src/components/EditorCanvas.tsx
import { useEffect, useRef, useState } from "react";
import { CKEditor } from "@ckeditor/ckeditor5-react";
import { editorClass, editorConfig } from "../lib/editorConfig";

type EditorCanvasProps = {
  initialData: string;
  onReady?: (editor: InstanceType<typeof editorClass>) => void;
  onChange?: (html: string) => void;
};

export function EditorCanvas({ initialData, onReady, onChange }: EditorCanvasProps) {
  const toolbarHostRef = useRef<HTMLDivElement | null>(null);
  const [mountedEditor, setMountedEditor] = useState<InstanceType<typeof editorClass> | null>(null);

  useEffect(() => {
    return () => {
      if (toolbarHostRef.current) {
        toolbarHostRef.current.innerHTML = "";
      }
    };
  }, []);

  return (
    <div className="studio-editor">
      <div ref={toolbarHostRef} className="studio-editor-toolbar" data-testid="ck5-toolbar-host" />
      <div className="studio-editor-paper" data-testid="ck5-paper">
        <CKEditor
          editor={editorClass as any}
          config={editorConfig}
          data={initialData}
          onReady={(editor) => {
            const toolbarElement = editor.ui.view.toolbar.element;
            if (toolbarElement && toolbarHostRef.current && !toolbarHostRef.current.contains(toolbarElement)) {
              toolbarHostRef.current.appendChild(toolbarElement);
            }
            setMountedEditor(editor as InstanceType<typeof editorClass>);
            onReady?.(editor as InstanceType<typeof editorClass>);
          }}
          onChange={(_, editor) => {
            onChange?.(editor.getData());
          }}
        />
      </div>
    </div>
  );
}
```

```tsx
// apps/ck5-studio/src/App.tsx
import { useState } from "react";
import { EditorCanvas } from "./components/EditorCanvas";

const INITIAL_HTML = "<h1>Untitled Editorial Concept</h1><p>Start writing here.</p>";

export default function App() {
  const [html, setHtml] = useState(INITIAL_HTML);
  return <EditorCanvas initialData={html} onChange={setHtml} />;
}
```

- [ ] **Step 4: Run build to verify native toolbar integration passes**

Run: `npm.cmd --prefix apps/ck5-studio run build`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/ck5-studio/src/lib/editorConfig.ts apps/ck5-studio/src/components/EditorCanvas.tsx apps/ck5-studio/src/App.tsx
git commit -m "feat(ck5-studio): mount decoupled ckeditor with native toolbar"
```

---

### Task 3: Build Editorial Shell And Reference Styling

**Files:**
- Create: `apps/ck5-studio/src/types.ts`
- Create: `apps/ck5-studio/src/components/AppShell.tsx`
- Create: `apps/ck5-studio/src/components/TopBar.tsx`
- Create: `apps/ck5-studio/src/components/LeftLibrary.tsx`
- Create: `apps/ck5-studio/src/components/RightPanel.tsx`
- Create: `apps/ck5-studio/src/styles/tokens.css`
- Create: `apps/ck5-studio/src/styles/base.css`
- Create: `apps/ck5-studio/src/styles/app.css`
- Modify: `apps/ck5-studio/src/main.tsx`
- Modify: `apps/ck5-studio/src/App.tsx`

- [ ] **Step 1: Write minimal structural smoke test for shell composition**

```ts
// apps/ck5-studio/src/__tests__/selectionSummary.test.ts
import { describe, expect, it } from "vitest";

describe("ck5 studio shell contract", () => {
  it("reserves top bar, left library, center canvas, right panel", () => {
    const regions = ["top-bar", "left-library", "editor-canvas", "right-panel"];
    expect(regions).toEqual(["top-bar", "left-library", "editor-canvas", "right-panel"]);
  });
});
```

- [ ] **Step 2: Run test to verify baseline suite works**

Run: `npm.cmd --prefix apps/ck5-studio run test`

Expected: PASS with trivial shell contract placeholder. This is suite bootstrap, not behavior proof.

- [ ] **Step 3: Add shell component structure**

```ts
// apps/ck5-studio/src/types.ts
export type LibraryItemKey = "text" | "heading" | "section" | "table" | "image";

export type SelectionSummary = {
  label: string;
  elementTag: string;
};
```

```tsx
// apps/ck5-studio/src/components/TopBar.tsx
export function TopBar() {
  return (
    <header className="studio-topbar" data-testid="top-bar">
      <div className="studio-brand">The Editorial Architect</div>
      <nav className="studio-nav">
        <a href="#" className="is-muted">Documents</a>
        <a href="#" className="is-active">Templates</a>
        <a href="#" className="is-muted">Publish</a>
      </nav>
      <div className="studio-actions">
        <button type="button" className="ghost-btn">Preview</button>
        <button type="button" className="primary-btn">Share</button>
      </div>
    </header>
  );
}
```

```tsx
// apps/ck5-studio/src/components/LeftLibrary.tsx
import type { LibraryItemKey } from "../types";

type LeftLibraryProps = {
  onInsert: (key: LibraryItemKey) => void;
  onImagePick: () => void;
};

export function LeftLibrary({ onInsert, onImagePick }: LeftLibraryProps) {
  return (
    <aside className="studio-left-panel" data-testid="left-library">
      <div className="panel-kicker">Library</div>
      <div className="panel-subtle">Intellectual Atelier</div>
      <button type="button" className="panel-cta" onClick={() => onInsert("text")}>+ New Document</button>
      <div className="library-group-label">Basic Blocks</div>
      <button type="button" className="library-tile" onClick={() => onInsert("text")}>Text</button>
      <button type="button" className="library-tile" onClick={onImagePick}>Media</button>
      <button type="button" className="library-tile" onClick={() => onInsert("table")}>Table</button>
      <button type="button" className="library-tile" onClick={() => onInsert("section")}>Section</button>
      <div className="library-group-label">Structural</div>
      <button type="button" className="library-row" onClick={() => onInsert("heading")}>Heading</button>
    </aside>
  );
}
```

```tsx
// apps/ck5-studio/src/components/RightPanel.tsx
import type { SelectionSummary } from "../types";

type RightPanelProps = {
  selection: SelectionSummary | null;
};

export function RightPanel({ selection }: RightPanelProps) {
  return (
    <aside className="studio-right-panel" data-testid="right-panel">
      <div className="panel-title-row">
        <h2>Properties</h2>
        <span className="panel-chip">{selection?.label ?? "DOCUMENT"}</span>
      </div>
      <div className="panel-meta">Element: {selection?.elementTag ?? "body"}</div>
      <div className="panel-section-title">Dimensions</div>
      <div className="panel-card"><span>Width</span><strong>100%</strong></div>
      <div className="panel-section-title">Background Surface</div>
      <div className="color-row">
        <span className="color-dot is-paper" />
        <span className="color-dot is-maroon" />
        <span className="color-dot is-charcoal" />
        <span className="color-dot is-blush" />
      </div>
      <div className="panel-section-title">Spacing</div>
      <div className="panel-card is-spacing-card">32 / 48 / 32 / 48</div>
      <button type="button" className="primary-btn panel-save">Save Template</button>
      <button type="button" className="ghost-link">Discard Changes</button>
    </aside>
  );
}
```

```tsx
// apps/ck5-studio/src/components/AppShell.tsx
import type { ReactNode } from "react";
import type { LibraryItemKey, SelectionSummary } from "../types";
import { TopBar } from "./TopBar";
import { LeftLibrary } from "./LeftLibrary";
import { RightPanel } from "./RightPanel";

type AppShellProps = {
  children: ReactNode;
  selection: SelectionSummary | null;
  onInsert: (key: LibraryItemKey) => void;
  onImagePick: () => void;
};

export function AppShell({ children, selection, onInsert, onImagePick }: AppShellProps) {
  return (
    <div className="studio-shell">
      <TopBar />
      <div className="studio-body">
        <LeftLibrary onInsert={onInsert} onImagePick={onImagePick} />
        <main className="studio-center" data-testid="editor-canvas">{children}</main>
        <RightPanel selection={selection} />
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Add reference-driven CSS tokens and base styles**

```css
/* apps/ck5-studio/src/styles/tokens.css */
:root {
  --studio-maroon-900: #570000;
  --studio-maroon-700: #800000;
  --studio-paper: #ffffff;
  --studio-shell: #f7f8fe;
  --studio-panel: #eef4ff;
  --studio-ink: #151c25;
  --studio-muted: #8b6f69;
  --studio-blush: #ffd7d1;
  --studio-radius-md: 0.75rem;
  --studio-radius-lg: 1rem;
  --studio-shadow-soft: 0 24px 48px rgba(21, 28, 37, 0.08);
}
```

```css
/* apps/ck5-studio/src/styles/base.css */
@import "@fontsource/inter/400.css";
@import "@fontsource/inter/500.css";
@import "@fontsource/inter/600.css";
@import "@fontsource/newsreader/400.css";
@import "@fontsource/newsreader/500.css";

html, body, #root {
  min-height: 100%;
  margin: 0;
}

body {
  font-family: "Inter", sans-serif;
  color: var(--studio-ink);
  background: var(--studio-shell);
}

button, input {
  font: inherit;
}
```

```css
/* apps/ck5-studio/src/styles/app.css */
.studio-shell { min-height: 100vh; background: var(--studio-shell); }
.studio-topbar {
  height: 80px;
  display: grid;
  grid-template-columns: 280px 1fr auto;
  align-items: center;
  padding: 0 28px;
  background: rgba(255, 255, 255, 0.78);
  backdrop-filter: blur(12px);
}
.studio-brand { font-family: "Newsreader", serif; font-size: 1.1rem; color: var(--studio-maroon-700); }
.studio-nav { display: flex; justify-content: center; gap: 44px; }
.studio-nav a { color: var(--studio-ink); text-decoration: none; }
.studio-nav a.is-active { color: var(--studio-maroon-700); border-bottom: 2px solid var(--studio-maroon-700); padding-bottom: 10px; }
.studio-body { display: grid; grid-template-columns: 340px minmax(720px, 1fr) 360px; min-height: calc(100vh - 80px); }
.studio-left-panel, .studio-right-panel { background: linear-gradient(180deg, #f7f8fe 0%, var(--studio-panel) 100%); padding: 28px 20px; }
.studio-center { background: #dfe8f9; display: flex; justify-content: center; padding: 32px 24px 56px; }
.studio-editor { width: min(820px, 100%); }
.studio-editor-toolbar .ck-toolbar { border: none !important; border-radius: var(--studio-radius-lg) !important; box-shadow: var(--studio-shadow-soft); }
.studio-editor-paper { margin-top: 20px; background: var(--studio-paper); min-height: 1120px; padding: 72px 88px; box-shadow: var(--studio-shadow-soft); }
.studio-editor-paper .ck-editor__editable_inline { min-height: 920px; border: none !important; box-shadow: none !important; font-size: 1rem; line-height: 1.6; }
.studio-editor-paper .ck-content h1, .studio-editor-paper .ck-content h2, .studio-editor-paper .ck-content h3 { font-family: "Newsreader", serif; }
.primary-btn { border: none; border-radius: 16px; color: white; background: linear-gradient(135deg, var(--studio-maroon-900), var(--studio-maroon-700)); padding: 14px 28px; }
```

- [ ] **Step 5: Wire styles and shell into app**

```tsx
// apps/ck5-studio/src/main.tsx
import "./styles/tokens.css";
import "./styles/base.css";
import "./styles/app.css";
```

```tsx
// apps/ck5-studio/src/App.tsx
import { useState } from "react";
import { AppShell } from "./components/AppShell";
import { EditorCanvas } from "./components/EditorCanvas";
import type { LibraryItemKey, SelectionSummary } from "./types";

const INITIAL_HTML = "<h1>Untitled Editorial Concept</h1><p>Start writing here.</p>";

export default function App() {
  const [html, setHtml] = useState(INITIAL_HTML);
  const [selection, setSelection] = useState<SelectionSummary | null>(null);

  function handleInsert(_key: LibraryItemKey) {}

  return (
    <AppShell selection={selection} onInsert={handleInsert} onImagePick={() => {}}>
      <EditorCanvas initialData={html} onChange={setHtml} />
    </AppShell>
  );
}
```

- [ ] **Step 6: Run build and visual dev check**

Run:
- `npm.cmd --prefix apps/ck5-studio run build`
- `npm.cmd --prefix apps/ck5-studio run dev`

Expected:
- build PASS
- app boots with top/left/center/right shell visible
- native toolbar still present

- [ ] **Step 7: Commit**

```bash
git add apps/ck5-studio/src/types.ts apps/ck5-studio/src/components/AppShell.tsx apps/ck5-studio/src/components/TopBar.tsx apps/ck5-studio/src/components/LeftLibrary.tsx apps/ck5-studio/src/components/RightPanel.tsx apps/ck5-studio/src/styles/tokens.css apps/ck5-studio/src/styles/base.css apps/ck5-studio/src/styles/app.css apps/ck5-studio/src/main.tsx apps/ck5-studio/src/App.tsx
git commit -m "feat(ck5-studio): add editorial shell and styling"
```

---

### Task 4: Add Palette Inserts And Local Image Handling

**Files:**
- Create: `apps/ck5-studio/src/lib/contentSnippets.ts`
- Create: `apps/ck5-studio/src/__tests__/contentSnippets.test.ts`
- Modify: `apps/ck5-studio/src/components/EditorCanvas.tsx`
- Modify: `apps/ck5-studio/src/App.tsx`

- [ ] **Step 1: Write failing snippet tests**

```ts
// apps/ck5-studio/src/__tests__/contentSnippets.test.ts
import { describe, expect, it } from "vitest";
import { snippetFor } from "../lib/contentSnippets";

describe("contentSnippets", () => {
  it("returns section snippet with visual header shell", () => {
    expect(snippetFor("section")).toContain("editorial-section");
  });

  it("returns starter table markup", () => {
    expect(snippetFor("table")).toContain("<table>");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm.cmd --prefix apps/ck5-studio run test -- --run src/__tests__/contentSnippets.test.ts`

Expected: FAIL because snippet helper does not exist.

- [ ] **Step 3: Create snippet helper**

```ts
// apps/ck5-studio/src/lib/contentSnippets.ts
import type { LibraryItemKey } from "../types";

export function snippetFor(key: Exclude<LibraryItemKey, "image">): string {
  switch (key) {
    case "text":
      return "<p>New paragraph block.</p>";
    case "heading":
      return "<h2>Framework</h2><p>Establishing trust through classic editorial structure.</p>";
    case "section":
      return `<section class="editorial-section"><h2>Section Title</h2><p><strong>Oct 24, 2023</strong> · Senior Editor · Draft</p></section>`;
    case "table":
      return `<table><thead><tr><th>Framework</th><th>Outcome</th></tr></thead><tbody><tr><td>Minimalist Cohesion</td><td>Reduced cognitive load for deep focus writing.</td></tr><tr><td>Typographic Authority</td><td>Establishing trust through classic editorial stems.</td></tr></tbody></table>`;
  }
}
```

- [ ] **Step 4: Expose insert API from editor**

```tsx
// apps/ck5-studio/src/components/EditorCanvas.tsx
import { forwardRef, useImperativeHandle } from "react";

export type EditorCanvasHandle = {
  insertHtml: (html: string) => void;
  insertImageFile: (file: File) => Promise<void>;
};

export const EditorCanvas = forwardRef<EditorCanvasHandle, EditorCanvasProps>(
  function EditorCanvas({ initialData, onReady, onChange }, ref) {
    const [mountedEditor, setMountedEditor] = useState<InstanceType<typeof editorClass> | null>(null);

    useImperativeHandle(ref, () => ({
      insertHtml(html: string) {
        if (!mountedEditor) return;
        const viewFragment = mountedEditor.data.processor.toView(html);
        const modelFragment = mountedEditor.data.toModel(viewFragment);
        mountedEditor.model.insertContent(modelFragment);
      },
      async insertImageFile(file: File) {
        if (!mountedEditor) return;
        const reader = new FileReader();
        const dataUrl = await new Promise<string>((resolve, reject) => {
          reader.onload = () => resolve(String(reader.result));
          reader.onerror = () => reject(reader.error);
          reader.readAsDataURL(file);
        });
        mountedEditor.model.change((writer) => {
          const imageElement = writer.createElement("imageBlock", { src: dataUrl });
          mountedEditor.model.insertContent(imageElement, mountedEditor.model.document.selection);
        });
      },
    }), [mountedEditor]);
```

- [ ] **Step 5: Wire left library insert actions**

```tsx
// apps/ck5-studio/src/App.tsx
import { useRef } from "react";
import { snippetFor } from "./lib/contentSnippets";
import { EditorCanvas, type EditorCanvasHandle } from "./components/EditorCanvas";

export default function App() {
  const editorRef = useRef<EditorCanvasHandle | null>(null);
  const hiddenImageInputRef = useRef<HTMLInputElement | null>(null);

  function handleInsert(key: LibraryItemKey) {
    if (key === "image") {
      hiddenImageInputRef.current?.click();
      return;
    }
    editorRef.current?.insertHtml(snippetFor(key));
  }

  return (
    <>
      <input
        ref={hiddenImageInputRef}
        hidden
        type="file"
        accept="image/*"
        onChange={(event) => {
          const file = event.target.files?.[0];
          if (file) {
            void editorRef.current?.insertImageFile(file);
          }
          event.currentTarget.value = "";
        }}
      />
      <AppShell selection={selection} onInsert={handleInsert} onImagePick={() => hiddenImageInputRef.current?.click()}>
        <EditorCanvas ref={editorRef} initialData={html} onChange={setHtml} />
      </AppShell>
    </>
  );
}
```

- [ ] **Step 6: Run tests and manual dev check**

Run:
- `npm.cmd --prefix apps/ck5-studio run test -- --run src/__tests__/contentSnippets.test.ts`
- `npm.cmd --prefix apps/ck5-studio run dev`

Expected:
- snippet test PASS
- clicking left library buttons inserts content
- picking image inserts local image into canvas

- [ ] **Step 7: Commit**

```bash
git add apps/ck5-studio/src/lib/contentSnippets.ts apps/ck5-studio/src/__tests__/contentSnippets.test.ts apps/ck5-studio/src/components/EditorCanvas.tsx apps/ck5-studio/src/App.tsx
git commit -m "feat(ck5-studio): add palette inserts and local image handling"
```

---

### Task 5: Add Selection Summary And Local Draft Restore

**Files:**
- Create: `apps/ck5-studio/src/lib/localDraft.ts`
- Create: `apps/ck5-studio/src/lib/selectionSummary.ts`
- Create: `apps/ck5-studio/src/__tests__/localDraft.test.ts`
- Create: `apps/ck5-studio/src/__tests__/selectionSummary.test.ts`
- Modify: `apps/ck5-studio/src/components/EditorCanvas.tsx`
- Modify: `apps/ck5-studio/src/App.tsx`

- [ ] **Step 1: Write failing utility tests**

```ts
// apps/ck5-studio/src/__tests__/localDraft.test.ts
import { describe, expect, it, vi } from "vitest";
import { loadDraft, saveDraft } from "../lib/localDraft";

describe("localDraft", () => {
  it("saves and loads html snapshot", () => {
    const storage = { getItem: vi.fn(), setItem: vi.fn() } as unknown as Storage;
    saveDraft(storage, "<p>Hello</p>");
    expect((storage.setItem as any).mock.calls[0][0]).toBe("ck5-studio:draft");
    expect((storage.setItem as any).mock.calls[0][1]).toContain("Hello");
  });

  it("returns null when no draft exists", () => {
    const storage = { getItem: vi.fn(() => null) } as unknown as Storage;
    expect(loadDraft(storage)).toBeNull();
  });
});
```

```ts
// apps/ck5-studio/src/__tests__/selectionSummary.test.ts
import { describe, expect, it } from "vitest";
import { summarizeSelectionTag } from "../lib/selectionSummary";

describe("selectionSummary", () => {
  it("maps heading to HEADING", () => {
    expect(summarizeSelectionTag("h2")).toEqual({ label: "HEADING", elementTag: "h2" });
  });

  it("maps paragraph to TEXT", () => {
    expect(summarizeSelectionTag("p")).toEqual({ label: "TEXT", elementTag: "p" });
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm.cmd --prefix apps/ck5-studio run test -- --run src/__tests__/localDraft.test.ts src/__tests__/selectionSummary.test.ts`

Expected: FAIL because helpers do not exist.

- [ ] **Step 3: Create local draft and selection summary helpers**

```ts
// apps/ck5-studio/src/lib/localDraft.ts
const STORAGE_KEY = "ck5-studio:draft";

export function saveDraft(storage: Storage, html: string) {
  storage.setItem(STORAGE_KEY, JSON.stringify({ html, savedAt: new Date().toISOString() }));
}

export function loadDraft(storage: Storage): string | null {
  const raw = storage.getItem(STORAGE_KEY);
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as { html?: string };
    return typeof parsed.html === "string" ? parsed.html : null;
  } catch {
    return null;
  }
}
```

```ts
// apps/ck5-studio/src/lib/selectionSummary.ts
import type { SelectionSummary } from "../types";

export function summarizeSelectionTag(tagName: string): SelectionSummary {
  const normalized = tagName.toLowerCase();
  if (normalized === "h1" || normalized === "h2" || normalized === "h3") return { label: "HEADING", elementTag: normalized };
  if (normalized === "table") return { label: "TABLE", elementTag: normalized };
  if (normalized === "img" || normalized === "figure") return { label: "IMAGE", elementTag: normalized };
  if (normalized === "section") return { label: "SECTION", elementTag: normalized };
  return { label: "TEXT", elementTag: normalized || "body" };
}
```

- [ ] **Step 4: Wire autosave and selection reporting**

```tsx
// apps/ck5-studio/src/components/EditorCanvas.tsx
type EditorCanvasProps = {
  initialData: string;
  onReady?: (editor: InstanceType<typeof editorClass>) => void;
  onChange?: (html: string) => void;
  onSelectionSummary?: (tagName: string) => void;
};

onReady={(editor) => {
  const toolbarElement = editor.ui.view.toolbar.element;
  if (toolbarElement && toolbarHostRef.current && !toolbarHostRef.current.contains(toolbarElement)) {
    toolbarHostRef.current.appendChild(toolbarElement);
  }
  setMountedEditor(editor as InstanceType<typeof editorClass>);
  editor.editing.view.document.on("selectionChange", () => {
    const nativeSelection = window.getSelection();
    const node = nativeSelection?.anchorNode instanceof Element
      ? nativeSelection.anchorNode
      : nativeSelection?.anchorNode?.parentElement;
    const tagName = node?.closest("h1,h2,h3,p,table,figure,section")?.tagName?.toLowerCase() ?? "body";
    onSelectionSummary?.(tagName);
  });
  onReady?.(editor as InstanceType<typeof editorClass>);
}}
```

```tsx
// apps/ck5-studio/src/App.tsx
import { loadDraft, saveDraft } from "./lib/localDraft";
import { summarizeSelectionTag } from "./lib/selectionSummary";

const initialHtml = typeof window !== "undefined" ? (loadDraft(window.localStorage) ?? INITIAL_HTML) : INITIAL_HTML;
const [html, setHtml] = useState(initialHtml);

function handleHtmlChange(nextHtml: string) {
  setHtml(nextHtml);
  if (typeof window !== "undefined") {
    saveDraft(window.localStorage, nextHtml);
  }
}

<EditorCanvas
  ref={editorRef}
  initialData={html}
  onChange={handleHtmlChange}
  onSelectionSummary={(tag) => setSelection(summarizeSelectionTag(tag))}
/>
```

- [ ] **Step 5: Run tests and manual dev check**

Run:
- `npm.cmd --prefix apps/ck5-studio run test -- --run src/__tests__/localDraft.test.ts src/__tests__/selectionSummary.test.ts`
- `npm.cmd --prefix apps/ck5-studio run dev`

Expected:
- utility tests PASS
- right panel label changes while moving selection
- reload restores local draft

- [ ] **Step 6: Commit**

```bash
git add apps/ck5-studio/src/lib/localDraft.ts apps/ck5-studio/src/lib/selectionSummary.ts apps/ck5-studio/src/__tests__/localDraft.test.ts apps/ck5-studio/src/__tests__/selectionSummary.test.ts apps/ck5-studio/src/components/EditorCanvas.tsx apps/ck5-studio/src/App.tsx
git commit -m "feat(ck5-studio): add selection summary and local draft restore"
```

---

### Task 6: Add Real Chromium Smoke And Visual Verification

**Files:**
- Create: `apps/ck5-studio/playwright.config.ts`
- Create: `apps/ck5-studio/tests/editorial-shell.spec.ts`
- Modify: `tasks/lessons.md`

- [ ] **Step 1: Write failing Playwright smoke spec**

```ts
// apps/ck5-studio/tests/editorial-shell.spec.ts
import { test, expect } from "@playwright/test";

test("editorial shell boots with native toolbar and side panels", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByTestId("top-bar")).toBeVisible();
  await expect(page.getByTestId("left-library")).toBeVisible();
  await expect(page.getByTestId("editor-canvas")).toBeVisible();
  await expect(page.getByTestId("right-panel")).toBeVisible();
  await expect(page.getByTestId("ck5-toolbar-host").locator(".ck-toolbar")).toBeVisible();
});

test("left palette inserts heading and table into editor", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Heading" }).click();
  await page.getByRole("button", { name: "Table" }).click();
  const paper = page.getByTestId("ck5-paper");
  await expect(paper.locator("h2")).toHaveCount(1);
  await expect(paper.locator("table")).toHaveCount(1);
});
```

- [ ] **Step 2: Run e2e to verify it fails before config exists**

Run: `npm.cmd --prefix apps/ck5-studio run e2e`

Expected: FAIL because Playwright config and web server settings do not exist.

- [ ] **Step 3: Add Playwright config with auto web server**

```ts
// apps/ck5-studio/playwright.config.ts
import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  use: {
    baseURL: "http://127.0.0.1:4175",
    headless: true,
  },
  webServer: {
    command: "npm.cmd run dev -- --host 127.0.0.1 --port 4175",
    url: "http://127.0.0.1:4175",
    reuseExistingServer: true,
    cwd: ".",
    timeout: 120000,
  },
});
```

- [ ] **Step 4: Extend smoke spec to capture screenshot**

```ts
test("captures editorial reference screenshot", async ({ page }) => {
  await page.goto("/");
  await page.screenshot({
    path: "../../tmp/visual-checks/ck5-studio-editorial-shell.png",
    fullPage: true,
  });
});
```

- [ ] **Step 5: Run browser verification loop**

Run:
- `npm.cmd --prefix apps/ck5-studio run e2e`
- `npm.cmd --prefix apps/ck5-studio run build`

Expected:
- Playwright PASS
- screenshot created at `tmp/visual-checks/ck5-studio-editorial-shell.png`
- build PASS

Manual browser checklist:
- native toolbar visible
- top/left/right shell visible
- heading insert works
- table insert works
- page looks close to reference style

- [ ] **Step 6: Record correction lesson**

```md
## Lesson N - Disposable spikes still need isolated app boundaries
Date: 2026-04-15 | Trigger: correction
Wrong:   Evaluating replacement editor inside production frontend would mix spike risk with live product structure.
Correct: CKEditor 5 ceiling spike lives in isolated `apps/ck5-studio/` app with local-only state and independent verification.
Rule:    Throwaway technology spikes must isolate dependencies, routes, and styling so evaluation stays reversible.
Layer:   process
```

- [ ] **Step 7: Commit**

```bash
git add apps/ck5-studio/playwright.config.ts apps/ck5-studio/tests/editorial-shell.spec.ts tasks/lessons.md tmp/visual-checks/ck5-studio-editorial-shell.png
git commit -m "test(ck5-studio): verify editorial spike in chromium"
```

---

## Self-Review

**1. Spec coverage**
- isolated disposable app in `apps/ck5-studio/`: Task 1
- native CKEditor toolbar: Task 2
- editorial shell matching reference: Task 3
- left block library with `Text`, `Heading`, `Section`, `Table`, `Image`: Task 4
- right properties shell: Task 3 + Task 5
- local-only state and optional `localStorage`: Task 5
- real Chromium verification with screenshots: Task 6

**2. Placeholder scan**
- No `TODO` / `TBD`
- each task names exact files
- each code-changing step includes concrete code
- each verification step includes exact commands

**3. Type consistency**
- insertion keys use one shared type: `LibraryItemKey`
- selection panel uses one shared type: `SelectionSummary`
- editor imperative API uses `EditorCanvasHandle`
- local draft key fixed as `ck5-studio:draft`

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-15-ck5-studio-visual-spike.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
