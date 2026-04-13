// @vitest-environment jsdom
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { DocumentBrowserEditorBundleResponse, DocumentListItem } from "../../../../lib.types";

const {
  getDocumentBrowserEditorBundleMock,
  saveDocumentBrowserContentMock,
  documentEditorHeaderMock,
  mddmEditorMock,
  mddmViewerMock,
} = vi.hoisted(() => ({
  getDocumentBrowserEditorBundleMock: vi.fn(),
  saveDocumentBrowserContentMock: vi.fn(),
  documentEditorHeaderMock: vi.fn(),
  mddmEditorMock: vi.fn(),
  mddmViewerMock: vi.fn(),
}));

vi.mock("../../../../api/documents", () => ({
  getDocumentBrowserEditorBundle: getDocumentBrowserEditorBundleMock,
  saveDocumentBrowserContent: saveDocumentBrowserContentMock,
}));

vi.mock("../DocumentEditorHeader", () => ({
  DocumentEditorHeader: (props: unknown) => {
    documentEditorHeaderMock(props);
    return <div data-testid="document-editor-header" />;
  },
}));

vi.mock("../../mddm-editor/MDDMEditor", () => ({
  MDDMEditor: (props: unknown) => {
    mddmEditorMock(props);
    return <div data-testid="mddm-editor-root" />;
  },
}));

vi.mock("../../mddm-editor/MDDMViewer", () => ({
  MDDMViewer: (props: unknown) => {
    mddmViewerMock(props);
    return <div data-testid="mddm-viewer-root" />;
  },
}));

vi.mock("../SaveBeforeExportDialog", () => ({
  SaveBeforeExportDialog: () => null,
}));

import { BrowserDocumentEditorView } from "../BrowserDocumentEditorView";

const sampleDocument: DocumentListItem = {
  documentId: "doc-123",
  title: "Documento teste",
  documentType: "type-1",
  documentProfile: "po",
  documentFamily: "family-1",
  ownerId: "user-1",
  businessUnit: "BU",
  department: "Operations",
  classification: "INTERNAL",
  status: "DRAFT",
  tags: [],
  createdAt: "2026-04-13T00:00:00Z",
  documentCode: "PO-001",
  processArea: "area-1",
  subject: "subject-1",
};

function sampleBundle(): DocumentBrowserEditorBundleResponse {
  return {
    document: sampleDocument,
    versions: [
      {
        documentId: sampleDocument.documentId,
        version: 1,
        contentHash: "hash-1",
        changeSummary: "Initial version",
        createdAt: "2026-04-13T00:00:00Z",
        renderer_pin: null,
      },
    ],
    governance: {
      profileCode: "po",
      workflowProfile: "default",
      reviewIntervalDays: 30,
      approvalRequired: false,
      retentionDays: 365,
      validityDays: 365,
    },
    templateSnapshot: {
      templateKey: "po-default",
      version: 1,
      profileCode: "po",
      schemaVersion: 1,
      editor: "mddm-blocknote",
      contentFormat: "mddm",
      body: "",
      definition: {},
    },
    body: JSON.stringify({ mddm_version: 1, template_ref: null, blocks: [] }),
    draftToken: "draft-1",
  };
}

describe("BrowserDocumentEditorView structure", () => {
  let host: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  beforeEach(() => {
    host = document.createElement("div");
    document.body.appendChild(host);
    root = createRoot(host);
    getDocumentBrowserEditorBundleMock.mockResolvedValue(sampleBundle());
  });

  afterEach(() => {
    act(() => root.unmount());
    host.remove();
    vi.clearAllMocks();
  });

  it("wires the editable branch through MDDMEditor with the loaded block-note payload", async () => {
    act(() => {
      root.render(<BrowserDocumentEditorView document={sampleDocument} onBack={vi.fn()} />);
    });

    await act(async () => {
      await Promise.resolve();
    });

    const viewport = host.querySelector('[data-testid="browser-editor-viewport"]');
    expect(viewport?.querySelector('[data-testid="document-editor-header"]')).not.toBeNull();
    expect(viewport?.querySelector('[data-testid="mddm-editor-root"]')).not.toBeNull();
    expect(documentEditorHeaderMock).toHaveBeenCalledTimes(1);
    expect(mddmEditorMock).toHaveBeenCalledTimes(1);
    expect(mddmViewerMock).not.toHaveBeenCalled();

    const editorProps = mddmEditorMock.mock.calls[0]?.[0] as
      | {
          initialContent?: unknown[];
          documentId?: string;
          readOnly?: boolean;
          theme?: unknown;
          onEditorReady?: unknown;
          onChange?: unknown;
        }
      | undefined;
    expect(editorProps?.documentId).toBe(sampleDocument.documentId);
    expect(editorProps?.readOnly).toBe(false);
    expect(Array.isArray(editorProps?.initialContent)).toBe(true);
    expect(editorProps?.initialContent).toHaveLength(0);
    expect((editorProps?.initialContent as { __mddm_envelope_meta__?: unknown } | undefined)?.__mddm_envelope_meta__).toBeDefined();
    expect(editorProps?.theme).toBeUndefined();
    expect(editorProps?.onEditorReady).toEqual(expect.any(Function));
    expect(editorProps?.onChange).toEqual(expect.any(Function));
  });

  it("wires the read-only branch through MDDMViewer when the document is published", async () => {
    const publishedDocument = { ...sampleDocument, status: "PUBLISHED" as const };

    act(() => {
      root.render(<BrowserDocumentEditorView document={publishedDocument} onBack={vi.fn()} />);
    });

    await act(async () => {
      await Promise.resolve();
    });

    const viewport = host.querySelector('[data-testid="browser-editor-viewport"]');
    expect(viewport?.querySelector('[data-testid="mddm-viewer-root"]')).not.toBeNull();
    expect(viewport?.querySelector('[data-testid="mddm-editor-root"]')).toBeNull();
    expect(documentEditorHeaderMock).toHaveBeenCalledTimes(1);
    expect(mddmViewerMock).toHaveBeenCalledTimes(1);
    expect(mddmEditorMock).not.toHaveBeenCalled();

    const viewerProps = mddmViewerMock.mock.calls[0]?.[0] as
      | {
          initialContent?: unknown[];
          documentId?: string;
          theme?: unknown;
        }
      | undefined;
    expect(viewerProps?.documentId).toBe(publishedDocument.documentId);
    expect(Array.isArray(viewerProps?.initialContent)).toBe(true);
    expect(viewerProps?.initialContent).toHaveLength(0);
    expect((viewerProps?.initialContent as { __mddm_envelope_meta__?: unknown } | undefined)?.__mddm_envelope_meta__).toBeDefined();
    expect(viewerProps?.theme).toBeUndefined();
  });
});
