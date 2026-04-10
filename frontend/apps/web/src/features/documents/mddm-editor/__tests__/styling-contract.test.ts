/// <reference types="node" />

import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

const workspaceRoot = process.cwd();

function readRepoFile(relativePath: string): string {
  return readFileSync(resolve(workspaceRoot, relativePath), "utf8");
}

function normalizeWhitespace(value: string): string {
  return value.replace(/\s+/g, " ").trim();
}

describe("MDDM styling contracts", () => {
  it("keeps FieldGroup structural-only", () => {
    const fieldGroupTsx = readRepoFile(
      "src/features/documents/mddm-editor/blocks/FieldGroup.tsx",
    );

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
    const css = readRepoFile(
      "src/features/documents/mddm-editor/mddm-editor-global.css",
    );
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
