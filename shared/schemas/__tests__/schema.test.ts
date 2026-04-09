import { describe, it, expect } from "vitest";
import { readFileSync, readdirSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import { validateMDDM } from "../validate";

const __dirname = dirname(fileURLToPath(import.meta.url));
const fixturesDir = join(__dirname, "..", "test-fixtures");

describe("MDDM Schema validation", () => {
  it("accepts optional section prop in section props", () => {
    const input = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "11111111-1111-1111-1111-111111111111",
          type: "section",
          props: { title: "7 - Indicadores", color: "#6b1f2a", locked: true, optional: true },
          children: [],
        },
      ],
    };

    const result = validateMDDM(input as any);
    expect(result.valid, JSON.stringify(result.errors)).toBe(true);
  });

  describe("valid fixtures", () => {
    const validDir = join(fixturesDir, "valid");
    for (const filename of readdirSync(validDir)) {
      it(`accepts ${filename}`, () => {
        const json = JSON.parse(readFileSync(join(validDir, filename), "utf8"));
        const result = validateMDDM(json);
        expect(result.valid, JSON.stringify(result.errors)).toBe(true);
      });
    }
  });

  describe("invalid fixtures", () => {
    const invalidDir = join(fixturesDir, "invalid");
    for (const filename of readdirSync(invalidDir)) {
      it(`rejects ${filename}`, () => {
        const json = JSON.parse(readFileSync(join(invalidDir, filename), "utf8"));
        const result = validateMDDM(json);
        expect(result.valid).toBe(false);
      });
    }
  });
});
