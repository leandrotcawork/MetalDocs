import { createRoot } from "react-dom/client";
import { act } from "react-dom/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { defaultLayoutTokens } from "../engine/layout-ir";
import { getEditorTokens } from "../engine/editor-tokens";

const editor = {};

vi.mock("@blocknote/react", () => ({
  useCreateBlockNote: vi.fn(() => editor),
  createReactBlockSpec: vi.fn((config: object, spec: object) => ({
    config,
    ...spec,
  })),
}));

vi.mock("@blocknote/mantine", () => ({
  BlockNoteView: () => null,
}));

vi.mock("../schema", () => ({
  mddmSchema: {},
}));

import { MDDMEditor } from "../MDDMEditor";

describe("MDDMEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("attaches theme tokens before calling onEditorReady", () => {
    const onEditorReady = vi.fn();
    const host = document.createElement("div");
    document.body.appendChild(host);
    const root = createRoot(host);

    act(() => {
      root.render(
        <MDDMEditor
          theme={{ accent: "#00ff00" }}
          onEditorReady={onEditorReady}
        />,
      );
    });

    expect(onEditorReady).toHaveBeenCalledTimes(1);
    expect(onEditorReady).toHaveBeenCalledWith(editor);
    expect(getEditorTokens(editor).theme.accent).toBe("#00ff00");
    expect(getEditorTokens(editor)).not.toBe(defaultLayoutTokens);

    act(() => {
      root.render(
        <MDDMEditor
          theme={{ accent: "#ff00ff" }}
          onEditorReady={onEditorReady}
        />,
      );
    });

    expect(onEditorReady).toHaveBeenCalledTimes(1);
    expect(getEditorTokens(editor).theme.accent).toBe("#ff00ff");

    act(() => {
      root.unmount();
    });

    host.remove();
  });
});
