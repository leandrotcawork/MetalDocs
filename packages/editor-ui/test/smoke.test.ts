import { describe, expect, it } from "vitest";
import * as EditorUI from "../src/index";

describe("editor-ui smoke", () => {
  it("imports module namespace", () => {
    expect(EditorUI).toBeDefined();
  });
});