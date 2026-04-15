// @vitest-environment jsdom
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { defaultLayoutTokens } from "../engine/layout-ir";
import { getEditorTokens } from "../engine/editor-tokens";

let tiptapDom: HTMLDivElement;
const {
  useCreateBlockNoteMock,
  uploadAttachmentMock,
  getAttachmentDownloadURLMock,
  blockNoteViewPropsMock,
} = vi.hoisted(() => ({
  useCreateBlockNoteMock: vi.fn(() => editor),
  uploadAttachmentMock: vi.fn(),
  getAttachmentDownloadURLMock: vi.fn(),
  blockNoteViewPropsMock: vi.fn(),
}));

const editor = {
  _tiptapEditor: {
    view: {
      get dom() {
        return tiptapDom;
      },
    },
  },
  focus: vi.fn(),
  setTextCursorPosition: vi.fn(),
  document: [] as unknown[],
};

vi.mock("@blocknote/react", () => ({
  useCreateBlockNote: useCreateBlockNoteMock,
  createReactBlockSpec: vi.fn((config: object, spec: object) => ({
    config,
    ...spec,
  })),
  BlockNoteViewEditor: () => null,
  FormattingToolbar: () => null,
  BasicTextStyleButton: () => null,
  BlockTypeSelect: () => null,
  ColorStyleButton: () => null,
  CreateLinkButton: () => null,
  NestBlockButton: () => null,
  UnnestBlockButton: () => null,
}));

vi.mock("@blocknote/mantine", () => ({
  BlockNoteView: ({ children, ...props }: { children?: import("react").ReactNode }) => {
    blockNoteViewPropsMock(props);
    return children ?? null;
  },
}));

vi.mock("../toolbar/MddmTextAlignButton", () => ({
  MddmTextAlignButton: () => null,
}));

vi.mock("../schema", () => ({
  mddmSchema: {},
}));

vi.mock("../../../../api/documents", () => ({
  uploadAttachment: uploadAttachmentMock,
  getAttachmentDownloadURL: getAttachmentDownloadURLMock,
}));

import { MDDMEditor } from "../MDDMEditor";

describe("MDDMEditor", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    tiptapDom = document.createElement("div");
  });

  it("mounts the formatting toolbar when the editor is editable", () => {
    const host = document.createElement("div");
    document.body.appendChild(host);
    const root = createRoot(host);

    act(() => {
      root.render(<MDDMEditor />);
    });

    const editorRoot = host.querySelector('[data-testid="mddm-editor-root"]');
    const toolbar = host.querySelector('[data-testid="mddm-editor-toolbar"]');
    const paper = host.querySelector('[data-testid="mddm-editor-paper"]');

    expect(editorRoot).not.toBeNull();
    expect(toolbar).not.toBeNull();
    expect(paper).not.toBeNull();
    expect(host.querySelector('[data-mddm-editor-root="true"]')).not.toBeNull();

    const props = blockNoteViewPropsMock.mock.calls[0]?.[0] as
      | { tableHandles?: boolean; editable?: boolean }
      | undefined;
    expect(props?.tableHandles).toBe(false);
    expect(props?.editable).toBe(true);

    act(() => {
      root.unmount();
    });

    host.remove();
  });

  it("omits the toolbar band in readOnly mode", () => {
    const host = document.createElement("div");
    document.body.appendChild(host);
    const root = createRoot(host);

    act(() => {
      root.render(<MDDMEditor readOnly />);
    });

    expect(host.querySelector('[data-testid="mddm-editor-root"]')).not.toBeNull();
    expect(host.querySelector('[data-testid="mddm-editor-toolbar"]')).toBeNull();
    expect(host.querySelector('[data-testid="mddm-editor-paper"]')).not.toBeNull();

    const props = blockNoteViewPropsMock.mock.calls[0]?.[0] as
      | { tableHandles?: boolean; editable?: boolean }
      | undefined;
    expect(props?.tableHandles).toBe(false);
    expect(props?.editable).toBe(false);

    act(() => {
      root.unmount();
    });

    host.remove();
  });

  it("attaches theme tokens before calling onEditorReady", () => {
    const readySnapshots: string[] = [];
    const host = document.createElement("div");
    document.body.appendChild(host);
    const root = createRoot(host);
    const onEditorReady = vi.fn((editorArg: unknown) => {
      const readyTokens = getEditorTokens(editorArg as object);
      readySnapshots.push(readyTokens.theme.accent);
      expect(readyTokens.theme.accent).toBe("#00ff00");
    });

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
    expect(readySnapshots).toEqual(["#00ff00"]);
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
    expect(readySnapshots).toEqual(["#00ff00"]);
    expect(getEditorTokens(editor).theme.accent).toBe("#ff00ff");

    act(() => {
      root.unmount();
    });

    host.remove();
  });

  it("applies pageSettings override to runtime page tokens", () => {
    const host = document.createElement("div");
    document.body.appendChild(host);
    const root = createRoot(host);

    act(() => {
      root.render(
        <MDDMEditor
          pageSettings={{
            marginTopMm: 10,
            marginRightMm: 30,
            marginBottomMm: 12,
            marginLeftMm: 35,
          }}
        />,
      );
    });

    const tokens = getEditorTokens(editor);
    expect(tokens.page.marginTopMm).toBe(10);
    expect(tokens.page.marginRightMm).toBe(30);
    expect(tokens.page.marginBottomMm).toBe(12);
    expect(tokens.page.marginLeftMm).toBe(35);
    expect(tokens.page.contentWidthMm).toBe(tokens.page.widthMm - 35 - 30);

    act(() => {
      root.unmount();
    });

    host.remove();
  });

  it("locks header cells at the DOM level", async () => {
    const host = document.createElement("div");
    document.body.appendChild(host);
    const root = createRoot(host);

    act(() => {
      root.render(<MDDMEditor />);
    });

    const initialHeader = document.createElement("th");
    initialHeader.contentEditable = "true";
    tiptapDom.appendChild(initialHeader);
    await act(async () => {
      await Promise.resolve();
    });
    expect(initialHeader.contentEditable).toBe("false");

    const dynamicHeader = document.createElement("th");
    dynamicHeader.contentEditable = "true";
    tiptapDom.appendChild(dynamicHeader);
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });
    expect(dynamicHeader.contentEditable).toBe("false");

    act(() => {
      root.unmount();
    });

    host.remove();
  });

  it("keeps unmarked table cells editable and locks cells marked by data-background-color", async () => {
    // This uses the closest stable DOM invariant exposed by BlockNote: template
    // label cells carry data-background-color, while regular value cells do not.
    const host = document.createElement("div");
    document.body.appendChild(host);
    const root = createRoot(host);

    act(() => {
      root.render(<MDDMEditor />);
    });

    const labelTd = document.createElement("td");
    labelTd.contentEditable = "true";
    labelTd.setAttribute("data-background-color", "gray");

    const valueTd = document.createElement("td");
    valueTd.contentEditable = "true";

    const labelTh = document.createElement("th");
    labelTh.contentEditable = "true";
    labelTh.setAttribute("data-background-color", "gray");

    tiptapDom.append(labelTd, valueTd, labelTh);

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    expect(labelTd.contentEditable).toBe("false");
    expect(valueTd.contentEditable).toBe("true");
    expect(labelTh.contentEditable).toBe("false");

    act(() => { root.unmount(); });
    host.remove();
  });

  it("locks an existing cell when data-background-color is added after mount", async () => {
    const host = document.createElement("div");
    document.body.appendChild(host);
    const root = createRoot(host);

    act(() => {
      root.render(<MDDMEditor />);
    });

    const delayedLabelTd = document.createElement("td");
    delayedLabelTd.contentEditable = "true";
    tiptapDom.append(delayedLabelTd);

    await act(async () => {
      await Promise.resolve();
    });
    expect(delayedLabelTd.contentEditable).toBe("true");

    delayedLabelTd.setAttribute("data-background-color", "gray");
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    expect(delayedLabelTd.contentEditable).toBe("false");

    act(() => {
      root.unmount();
    });
    host.remove();
  });

  it("provides BlockNote attachment hooks when documentId is present", async () => {
    uploadAttachmentMock.mockResolvedValue({ attachmentId: "att-123" });
    getAttachmentDownloadURLMock.mockResolvedValue({
      attachmentId: "att-123",
      downloadUrl: "https://signed.example/att-123",
      expiresAt: "2026-04-13T10:00:00Z",
    });

    const host = document.createElement("div");
    document.body.appendChild(host);
    const root = createRoot(host);

    act(() => {
      root.render(<MDDMEditor documentId="doc-123" />);
    });

    const call = useCreateBlockNoteMock.mock.calls[0] as unknown as [unknown] | undefined;
    expect(call).toBeDefined();
    if (!call) {
      throw new Error("Expected useCreateBlockNote to be called.");
    }
    const options = call[0] as {
      uploadFile?: (file: File) => Promise<string>;
      resolveFileUrl?: (url: string) => Promise<string>;
    };
    expect(options.uploadFile).toBeTypeOf("function");
    expect(options.resolveFileUrl).toBeTypeOf("function");
    const uploadFile = options.uploadFile!;
    const resolveFileUrl = options.resolveFileUrl!;

    const file = new File(["image"], "diagram.png", { type: "image/png" });
    await expect(uploadFile(file)).resolves.toBe(
      "/api/v1/documents/doc-123/attachments/att-123/download-url",
    );
    expect(uploadAttachmentMock).toHaveBeenCalledWith("doc-123", file);

    await expect(
      resolveFileUrl("/api/v1/documents/doc-123/attachments/att-123/download-url"),
    ).resolves.toBe("https://signed.example/att-123");
    expect(getAttachmentDownloadURLMock).toHaveBeenCalledWith("doc-123", "att-123");

    await expect(resolveFileUrl("https://cdn.example/image.png")).resolves.toBe(
      "https://cdn.example/image.png",
    );

    act(() => {
      root.unmount();
    });

    host.remove();
  });

  it("omits BlockNote attachment hooks without a persisted documentId", () => {
    const host = document.createElement("div");
    document.body.appendChild(host);
    const root = createRoot(host);

    act(() => {
      root.render(<MDDMEditor />);
    });

    const call = useCreateBlockNoteMock.mock.calls[0] as unknown as [unknown] | undefined;
    expect(call).toBeDefined();
    if (!call) {
      throw new Error("Expected useCreateBlockNote to be called.");
    }
    const options = call[0] as {
      uploadFile?: (file: File) => Promise<string>;
      resolveFileUrl?: (url: string) => Promise<string>;
    };
    expect(options.uploadFile).toBeUndefined();
    expect(options.resolveFileUrl).toBeUndefined();

    act(() => {
      root.unmount();
    });

    host.remove();
  });

});
