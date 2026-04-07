import { describe, it, expect } from "vitest";
import { readFileSync, readdirSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import { validateMDDM } from "../validate";

const __dirname = dirname(fileURLToPath(import.meta.url));
const fixturesDir = join(__dirname, "..", "test-fixtures");

describe("MDDM Schema validation", () => {
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
