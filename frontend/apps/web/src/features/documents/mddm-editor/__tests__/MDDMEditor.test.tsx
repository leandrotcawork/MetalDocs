// @vitest-environment jsdom
import { createRoot } from "react-dom/client";
import { act } from "react-dom/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { defaultLayoutTokens } from "../engine/layout-ir";
import { getEditorTokens } from "../engine/editor-tokens";

let tiptapDom: HTMLDivElement;
const {
  useCreateBlockNoteMock,
  uploadAttachmentMock,
  getAttachmentDownloadURLMock,
} = vi.hoisted(() => ({
  useCreateBlockNoteMock: vi.fn(() => editor),
  uploadAttachmentMock: vi.fn(),
  getAttachmentDownloadURLMock: vi.fn(),
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
  FilePanelController: () => null,
  FormattingToolbar: () => null,
  getFormattingToolbarItems: () => [],
}));

vi.mock("@blocknote/mantine", () => ({
  BlockNoteView: ({ children }: { children?: import("react").ReactNode }) => children ?? null,
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
