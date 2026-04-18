import { describe, expect, it } from "vitest";
import * as Tokens from "../src/index";

describe("shared-tokens smoke", () => {
  it("imports module namespace", () => {
    expect(Tokens).toBeDefined();
  });
});