import { describe, it, expect } from "vitest";
import {
  safeParse,
  expectString,
  expectBoolean,
  expectNumber,
  stripUndefined,
  resolveThemeRef,
} from "../codec-utils";

describe("safeParse", () => {
  it("parses valid JSON", () => {
    expect(safeParse('{"a":1}', {})).toEqual({ a: 1 });
  });

  it("returns fallback on invalid JSON", () => {
    expect(safeParse("not json", { fallback: true })).toEqual({ fallback: true });
  });

  it("returns fallback on empty string", () => {
    expect(safeParse("", {})).toEqual({});
  });
});

describe("expectString", () => {
  it("returns string values", () => {
    expect(expectString("hello")).toBe("hello");
  });

  it("returns undefined for non-strings", () => {
    expect(expectString(42)).toBeUndefined();
    expect(expectString(null)).toBeUndefined();
    expect(expectString(undefined)).toBeUndefined();
  });
});

describe("expectBoolean", () => {
  it("returns boolean values", () => {
    expect(expectBoolean(true, false)).toBe(true);
  });

  it("returns default for non-booleans", () => {
    expect(expectBoolean("yes", false)).toBe(false);
    expect(expectBoolean(undefined, true)).toBe(true);
  });
});

describe("expectNumber", () => {
  it("returns number values", () => {
    expect(expectNumber(42, 0)).toBe(42);
  });

  it("returns default for non-numbers", () => {
    expect(expectNumber("42", 0)).toBe(0);
    expect(expectNumber(undefined, 10)).toBe(10);
  });
});

describe("stripUndefined", () => {
  it("removes undefined keys", () => {
    expect(stripUndefined({ a: 1, b: undefined, c: "x" })).toEqual({ a: 1, c: "x" });
  });
});

describe("resolveThemeRef", () => {
  const theme = { accent: "#6b1f2a", accentLight: "#f9f3f3", accentDark: "#3e1018", accentBorder: "#dfc8c8" };

  it("resolves theme.accent", () => {
    expect(resolveThemeRef("theme.accent", theme)).toBe("#6b1f2a");
  });

  it("resolves theme.accentLight", () => {
    expect(resolveThemeRef("theme.accentLight", theme)).toBe("#f9f3f3");
  });

  it("returns non-theme strings as-is", () => {
    expect(resolveThemeRef("#ff0000", theme)).toBe("#ff0000");
  });

  it("returns undefined for undefined input", () => {
    expect(resolveThemeRef(undefined, theme)).toBeUndefined();
  });
});
