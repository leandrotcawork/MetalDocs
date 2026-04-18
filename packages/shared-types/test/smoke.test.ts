import { describe, expect, it } from "vitest";
import * as Types from "../src/index";

describe("shared-types smoke", () => {
  it("imports module namespace", () => {
    expect(Types).toBeDefined();
  });
});