// @vitest-environment jsdom
import { createRoot } from "react-dom/client";
import { act } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const {
  useTemplateDraftMock,
  useTemplatesStoreMock,
  mddmEditorMock,
  validationPanelMock,
  strippedFieldsBannerMock,
} = vi.hoisted(() => ({
  useTemplateDraftMock: vi.fn(),
  useTemplatesStoreMock: vi.fn(),
  mddmEditorMock: vi.fn(),
  validationPanelMock: vi.fn(),
  strippedFieldsBannerMock: vi.fn(),
}));

vi.mock("../useTemplateDraft", () => ({
  useTemplateDraft: useTemplateDraftMock,
}));

vi.mock("../../../store/templates.store", () => ({
  useTemplatesStore: useTemplatesStoreMock,
}));

vi.mock("../../documents/mddm-editor/MDDMEditor", () => ({
  MDDMEditor: (props: unknown) => {
    mddmEditorMock(props);
    return <div data-testid="mock-mddm-editor" />;
  },
}));

vi.mock("../ValidationPanel", () => ({
  ValidationPanel: (props: unknown) => {
    validationPanelMock(props);
    return <div data-testid="mock-validation-panel" />;
  },
}));

vi.mock("../StrippedFieldsBanner", () => ({
  StrippedFieldsBanner: (props: unknown) => {
    strippedFieldsBannerMock(props);
    return <div data-testid="mock-stripped-fields-banner" />;
  },
}));

import { TemplateEditorView } from "../TemplateEditorView";

describe("TemplateEditorView layout", () => {
  let host: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  beforeEach(() => {
    host = document.createElement("div");
    document.body.appendChild(host);
    root = createRoot(host);

    useTemplateDraftMock.mockReturnValue({
      draft: {
        templateKey: "template-1",
        name: "Template 1",
        status: "draft",
        lockVersion: 3,
        hasStrippedFields: false,
        blocks: [],
      },
      isLoading: false,
      error: null,
      saveDraft: vi.fn(),
      publish: vi.fn(),
      discardDraft: vi.fn(),
      replaceDraft: vi.fn(),
    });

    useTemplatesStoreMock.mockReturnValue({
      isDirty: false,
      markDirty: vi.fn(),
      markClean: vi.fn(),
      clearTemplate: vi.fn(),
      validationErrors: [],
      setValidationErrors: vi.fn(),
      selectedBlockId: null,
      setSelectedBlock: vi.fn(),
    });
  });

  afterEach(() => {
    act(() => {
      root.unmount();
    });
    host.remove();
    vi.clearAllMocks();
  });

  it("exposes the dedicated layout shell, document pane, and sidebar rail", () => {
    act(() => {
      root.render(<TemplateEditorView profileCode="PO" templateKey="template-1" />);
    });

    const layout = host.querySelector('[data-testid="template-editor-layout"]');
    const workspace = host.querySelector('[data-testid="template-editor-sidebars"]');
    const documentPane = host.querySelector('[data-testid="template-editor-document-pane"]');

    expect(layout).not.toBeNull();
    expect(workspace).not.toBeNull();
    expect(documentPane).not.toBeNull();

    expect(layout?.contains(workspace as Node)).toBe(true);
    expect(workspace?.contains(documentPane as Node)).toBe(true);
    expect(documentPane?.querySelector('[data-testid="mock-mddm-editor"]')).not.toBeNull();
    expect(workspace?.querySelector('[data-testid="block-palette"]')).not.toBeNull();
    expect(workspace?.querySelector('[data-testid="property-sidebar"]')).not.toBeNull();
  });

  it("renders compact template actions instead of large workspace CTAs", () => {
    act(() => {
      root.render(<TemplateEditorView profileCode="po" templateKey="tpl-ux" />);
    });

    const bar = host.querySelector('[data-testid="metadata-bar"]');

    expect(bar).not.toBeNull();
    expect(bar?.getAttribute("data-density")).toBe("compact");
    expect(bar?.querySelector('[data-testid="template-preview-docx-btn"]')).not.toBeNull();
    expect(bar?.querySelector('[data-testid="template-discard-btn"]')).not.toBeNull();
    expect(bar?.querySelector('[data-testid="template-save-btn"]')).not.toBeNull();
    expect(bar?.querySelector('[data-testid="template-publish-btn"]')).not.toBeNull();
  });

  it("renders readable editing side panels", () => {
    act(() => {
      root.render(<TemplateEditorView profileCode="po" templateKey="tpl-ux" />);
    });

    const palette = host.querySelector('[data-testid="block-palette"]');
    const sidebar = host.querySelector('[data-testid="property-sidebar"]');

    expect(palette).not.toBeNull();
    expect(sidebar).not.toBeNull();
    expect(palette?.getAttribute("data-contrast")).toBe("high");
    expect(sidebar?.getAttribute("data-contrast")).toBe("high");
  });

  it("passes parsed page settings from draft.meta to MDDMEditor", () => {
    useTemplateDraftMock.mockReturnValue({
      draft: {
        templateKey: "template-1",
        name: "Template 1",
        status: "draft",
        lockVersion: 3,
        hasStrippedFields: false,
        blocks: [],
        meta: {
          page: {
            marginTopMm: 12,
            marginRightMm: 18,
            marginBottomMm: 22,
            marginLeftMm: 27,
          },
        },
      },
      isLoading: false,
      error: null,
      saveDraft: vi.fn(),
      publish: vi.fn(),
      discardDraft: vi.fn(),
      replaceDraft: vi.fn(),
    });

    act(() => {
      root.render(<TemplateEditorView profileCode="PO" templateKey="template-1" />);
    });

    const lastCall = mddmEditorMock.mock.calls.at(-1)?.[0] as { pageSettings?: unknown } | undefined;
    expect(lastCall?.pageSettings).toEqual({
      marginTopMm: 12,
      marginRightMm: 18,
      marginBottomMm: 22,
      marginLeftMm: 27,
    });
  });
});
