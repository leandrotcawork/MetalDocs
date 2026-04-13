/// <reference types="node" />

import { existsSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

// Resolve paths relative to this test file so the tests work regardless of
// which directory vitest is invoked from (app dir, repo root, or CI).
const __dirname = dirname(fileURLToPath(import.meta.url));
const mddmEditorDir = resolve(__dirname, "..");

function readRepoFile(relativePath: string): string {
  return readFileSync(resolve(mddmEditorDir, relativePath), "utf8");
}

function normalizeWhitespace(value: string): string {
  return value.replace(/\s+/g, " ").trim();
}

describe("MDDM styling contracts", () => {
  it("removes deprecated field block specs and component files", () => {
    const schema = readRepoFile("schema.ts");
    const fieldTsxPath = resolve(mddmEditorDir, "blocks/Field.tsx");
    const fieldCssPath = resolve(mddmEditorDir, "blocks/Field.module.css");
    const fieldGroupTsxPath = resolve(mddmEditorDir, "blocks/FieldGroup.tsx");
    const fieldGroupCssPath = resolve(mddmEditorDir, "blocks/FieldGroup.module.css");

    expect(schema).not.toContain("./blocks/Field");
    expect(schema).not.toContain("./blocks/FieldGroup");
    expect(schema).not.toContain("field: Field()");
    expect(schema).not.toContain("fieldGroup: FieldGroup()");
    expect(existsSync(fieldTsxPath)).toBe(false);
    expect(existsSync(fieldCssPath)).toBe(false);
    expect(existsSync(fieldGroupTsxPath)).toBe(false);
    expect(existsSync(fieldGroupCssPath)).toBe(false);
  });

  it("keeps the global bridge CSS free of removed field selectors", () => {
    const css = readRepoFile("mddm-editor-global.css");
    const normalizedCss = normalizeWhitespace(css);

    expect(normalizedCss).not.toContain(".react-renderer.node-fieldGroup + .bn-block-group");
    expect(normalizedCss).not.toContain('[data-content-type="fieldGroup"] > *');
    expect(normalizedCss).not.toContain('[data-content-type="field"] > *');
    expect(normalizedCss).not.toContain(".react-renderer.node-fieldGroup:has([data-columns]) + .bn-block-group");
    expect(normalizedCss).not.toContain('[data-content-type="fieldGroup"] > .bn-block-outer > .bn-block > .bn-side-menu');
    expect(normalizedCss).toContain(
      '[data-content-type="section"] > .bn-block-outer > .bn-block > .bn-side-menu',
    );
    expect(normalizedCss).toContain(
      '[data-content-type="repeatable"] > .bn-block-outer > .bn-block > .bn-side-menu',
    );
  });
});
