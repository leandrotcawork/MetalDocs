import { describe, expect, it } from "vitest";
import { defaultLayoutTokens } from "../layout-ir";
import { getEditorTokens, setEditorTokens } from "../editor-tokens";

describe("editor tokens", () => {
  it("getEditorTokens returns defaultLayoutTokens when none set", () => {
    const editor = {};

    expect(getEditorTokens(editor)).toBe(defaultLayoutTokens);
  });

  it("setEditorTokens stores custom tokens and getEditorTokens returns custom object", () => {
    const editor = {};
    const tokens = {
      ...defaultLayoutTokens,
      theme: {
        ...defaultLayoutTokens.theme,
        accent: "#ff0000",
      },
    };

    setEditorTokens(editor, tokens);

    expect(getEditorTokens(editor)).toBe(tokens);
    expect(getEditorTokens(editor).theme.accent).toBe("#ff0000");
  });
});
