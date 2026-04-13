/// <reference types="node" />

import { readFileSync } from "node:fs";
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
  it("keeps FieldGroup structural-only", () => {
    const fieldGroupTsx = readRepoFile("blocks/FieldGroup.tsx");

    expect(fieldGroupTsx).toMatch(
      /render:\s*\(props\)\s*=>\s*\(\s*<div\b[^>]*\/>\s*\)/s,
    );
    expect(fieldGroupTsx).toContain('data-mddm-block="fieldGroup"');
    expect(fieldGroupTsx).toContain("data-columns={props.block.props.columns}");
    expect(fieldGroupTsx).toContain("data-locked={props.block.props.locked}");
    expect(fieldGroupTsx).not.toContain("Field Group");
    expect(fieldGroupTsx).not.toContain("coluna(s)");
  });

  it("keeps explicit side-menu hide selectors in the global bridge CSS", () => {
    const css = readRepoFile("mddm-editor-global.css");
    const normalizedCss = normalizeWhitespace(css);

    expect(normalizedCss).toContain(
      '[data-content-type="section"] > .bn-block-outer > .bn-block > .bn-side-menu',
    );
    expect(normalizedCss).toContain(
      '[data-content-type="fieldGroup"] > .bn-block-outer > .bn-block > .bn-side-menu',
    );
    expect(normalizedCss).toContain(
      '[data-content-type="repeatable"] > .bn-block-outer > .bn-block > .bn-side-menu',
    );
  });
});
