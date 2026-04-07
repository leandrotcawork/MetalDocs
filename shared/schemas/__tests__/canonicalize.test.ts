import { describe, it, expect } from "vitest";
import { readFileSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import { canonicalizeMDDM } from "../canonicalize";

const __dirname = dirname(fileURLToPath(import.meta.url));
const fixturesDir = join(__dirname, "..", "test-fixtures", "canonical");

describe("canonicalizeMDDM", () => {
  it("produces canonical output for mixed-order input", () => {
    const input = JSON.parse(readFileSync(join(fixturesDir, "input-mixed-order.json"), "utf8"));
    const expected = JSON.parse(readFileSync(join(fixturesDir, "output-mixed-order.json"), "utf8"));
    const actual = canonicalizeMDDM(input);
    expect(JSON.stringify(actual)).toBe(JSON.stringify(expected));
  });
});
