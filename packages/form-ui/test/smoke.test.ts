import { describe, expect, it } from "vitest";
import * as FormUI from "../src/index";

describe("form-ui smoke", () => {
  it("imports module namespace", () => {
    expect(typeof FormUI).toBe("object");
  });
});
