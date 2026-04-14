/// <reference types="node" />

import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const __dirname = dirname(fileURLToPath(import.meta.url));
const mddmEditorDir = resolve(__dirname, "..");
const browserEditorDir = resolve(mddmEditorDir, "../browser-editor");

function readUtf8(path: string): string {
  return readFileSync(path, "utf8");
}

function normalize(value: string): string {
  return value.replace(/\s+/g, " ").trim();
}

function ruleBody(css: string, selector: string): string {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const match = css.match(new RegExp(`${escapedSelector}\\s*\\{([^}]*)\\}`));
  return normalize(match?.[1] ?? "");
}

describe("editor shell contracts", () => {
  it("keeps the browser editor viewport contract explicit", () => {
    const tsx = normalize(readUtf8(resolve(browserEditorDir, "BrowserDocumentEditorView.tsx")));
    const css = normalize(readUtf8(resolve(browserEditorDir, "BrowserDocumentEditorView.module.css")));
    const surfaceCss = ruleBody(css, ".surface");
    const viewportCss = ruleBody(css, ".editorViewport");
    const errorBannerCss = ruleBody(css, ".errorBanner");

    expect(tsx).toContain('data-testid="browser-editor-viewport"');
    expect(css).toContain(".editorViewport");
    expect(surfaceCss).toContain("padding: 0");
    expect(surfaceCss).not.toMatch(/overflow-[xy]\s*:/);
    expect(surfaceCss).not.toMatch(/overflow\s*:\s*(auto|scroll)/);
    expect(viewportCss).toContain("isolation: isolate");
    expect(viewportCss).toContain("padding: 0");
    expect(viewportCss).toContain("overflow: visible");
    expect(viewportCss).not.toMatch(/overflow-[xy]\s*:/);
    expect(viewportCss).not.toMatch(/overflow\s*:\s*(auto|scroll)/);
    expect(viewportCss).not.toContain("overflow: clip");
    expect(errorBannerCss).toContain("margin: 0.75rem 0.75rem 0");
  });

  it("keeps the editor root scoped and suppresses structural table handles", () => {
    const tsx = normalize(readUtf8(resolve(mddmEditorDir, "MDDMEditor.tsx")));
    const css = normalize(readUtf8(resolve(mddmEditorDir, "mddm-editor-global.css")));

    expect(tsx).toContain('data-mddm-editor-root="true"');
    expect(css).toContain('[data-mddm-editor-root="true"] .bn-container');
    expect(css).toContain('[data-mddm-editor-root="true"] .bn-table-handle');
    expect(css).toContain('[data-mddm-editor-root="true"] .bn-table-handle-menu');
    expect(css).toContain('[data-mddm-editor-root="true"] .bn-table-cell-handle');
  });
});
