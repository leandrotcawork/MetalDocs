import React from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useTemplateDraft } from "../useTemplateDraft";
import { useTemplatesStore } from "../../../store/templates.store";

const navigateMock = vi.fn();

vi.mock("react-router-dom", () => ({
  useNavigate: () => navigateMock,
}));

vi.mock("../../../api/templates", async () => {
  const actual = await vi.importActual<typeof import("../../../api/templates")>("../../../api/templates");
  return {
    ...actual,
    getTemplate: vi.fn(),
    editPublished: vi.fn(),
    saveDraft: vi.fn(),
    publishTemplate: vi.fn(),
    discardDraft: vi.fn(),
  };
});

type HarnessProps = {
  templateKey: string;
  blocks: unknown;
  replacementDraft?: Parameters<ReturnType<typeof useTemplateDraft>["replaceDraft"]>[0];
};

function Harness({ templateKey, blocks, replacementDraft }: HarnessProps) {
  const state = useTemplateDraft({ templateKey });

  return (
    <div>
      <div data-testid="draft-key">{state.draft?.templateKey ?? ""}</div>
      <div data-testid="draft-status">{state.draft?.status ?? ""}</div>
      <div data-testid="lock-version">{state.draft?.lockVersion ?? 0}</div>
      <div data-testid="has-stripped-fields">{String(state.draft?.hasStrippedFields ?? false)}</div>
      <button data-testid="publish" type="button" onClick={() => void state.publish(blocks)}>
        publish
      </button>
      <button data-testid="save" type="button" onClick={() => void state.saveDraft(blocks)}>
        save
      </button>
      <button
        data-testid="update-meta"
        type="button"
        onClick={() =>
          state.updateDraftMeta(() => ({
            page: {
              marginTopMm: 31,
              marginRightMm: 22,
              marginBottomMm: 33,
              marginLeftMm: 24,
            },
          }))
        }
      >
        update-meta
      </button>
      <button
        data-testid="replace-draft"
        type="button"
        onClick={() => {
          if (replacementDraft) {
            state.replaceDraft(replacementDraft);
          }
        }}
      >
        replace
      </button>
    </div>
  );
}

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

async function flush() {
  await act(async () => {
    await Promise.resolve();
  });
}

beforeEach(() => {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  navigateMock.mockReset();
  useTemplatesStore.getState().clearTemplate();
});

afterEach(() => {
  act(() => {
    root.unmount();
  });
  container.remove();
  vi.clearAllMocks();
  useTemplatesStore.getState().clearTemplate();
});

describe("useTemplateDraft", () => {
  it("creates an editable draft when loading a published template", async () => {
    const { getTemplate, editPublished } = await import("../../../api/templates");

    vi.mocked(getTemplate).mockResolvedValueOnce({
      templateKey: "tmpl-published",
      version: 3,
      profileCode: "po",
      name: "Published template",
      status: "published",
    });
    vi.mocked(editPublished).mockResolvedValueOnce({
      templateKey: "tmpl-published",
      profileCode: "po",
      name: "Published template",
      status: "draft",
      lockVersion: 7,
      hasStrippedFields: false,
      blocks: [],
      updatedAt: "2026-04-14T00:00:00Z",
    });

    act(() => {
      root.render(<Harness templateKey="tmpl-published" blocks={[]} />);
    });
    await flush();
    await flush();

    expect(editPublished).toHaveBeenCalledWith("tmpl-published");
    expect(container.querySelector('[data-testid="draft-status"]')?.textContent).toBe("draft");
    expect(container.querySelector('[data-testid="lock-version"]')?.textContent).toBe("7");
  });

  it("saves the latest editor blocks before publishing", async () => {
    const { getTemplate, saveDraft, publishTemplate } = await import("../../../api/templates");

    vi.mocked(getTemplate).mockResolvedValueOnce({
      templateKey: "tmpl-draft",
      profileCode: "po",
      name: "Draft template",
      status: "draft",
      lockVersion: 1,
      hasStrippedFields: false,
      blocks: [],
      meta: { page: { marginTopMm: 25 } },
      updatedAt: "2026-04-14T00:00:00Z",
    });
    vi.mocked(saveDraft).mockResolvedValueOnce({
      templateKey: "tmpl-draft",
      profileCode: "po",
      name: "Draft template",
      status: "draft",
      lockVersion: 2,
      hasStrippedFields: false,
      blocks: [{
        id: "section-1",
        type: "section",
        props: {
          title: "Header",
          caps: { locked: true, removable: false, reorderable: false },
        },
      }],
      updatedAt: "2026-04-14T00:00:01Z",
    });
    vi.mocked(publishTemplate).mockResolvedValueOnce({
      templateKey: "tmpl-draft",
      version: 1,
      profileCode: "po",
      name: "Draft template",
      status: "published",
    });

    act(() => {
      root.render(
        <Harness
          templateKey="tmpl-draft"
          blocks={[{
            id: "section-1",
            type: "section",
            props: {
              title: "Header",
              caps: { locked: true, removable: false, reorderable: false },
            },
          }]}
        />,
      );
    });
    await flush();

    const publishButton = container.querySelector('[data-testid="publish"]') as HTMLButtonElement | null;
    expect(publishButton).toBeTruthy();

    await act(async () => {
      publishButton?.click();
      await Promise.resolve();
    });

    expect(saveDraft).toHaveBeenCalledWith("tmpl-draft", {
      blocks: [{
        id: "section-1",
        type: "section",
        props: {
          title: "Header",
          caps: { locked: true, removable: false, reorderable: false },
        },
      }],
      meta: { page: { marginTopMm: 25 } },
      lockVersion: 1,
    });
    expect(publishTemplate).toHaveBeenCalledWith("tmpl-draft", 2);
    expect(navigateMock).toHaveBeenCalledWith(-1);
  });

  it("sends current meta when saving a draft", async () => {
    const { getTemplate, saveDraft } = await import("../../../api/templates");

    vi.mocked(getTemplate).mockResolvedValueOnce({
      templateKey: "tmpl-save",
      profileCode: "po",
      name: "Draft template",
      status: "draft",
      lockVersion: 3,
      hasStrippedFields: false,
      blocks: [],
      meta: { page: { marginLeftMm: 22, marginRightMm: 20 } },
      updatedAt: "2026-04-14T00:00:00Z",
    });
    vi.mocked(saveDraft).mockResolvedValueOnce({
      templateKey: "tmpl-save",
      profileCode: "po",
      name: "Draft template",
      status: "draft",
      lockVersion: 4,
      hasStrippedFields: false,
      blocks: [{ id: "p-1", type: "paragraph", content: "abc" }],
      meta: { page: { marginLeftMm: 22, marginRightMm: 20 } },
      updatedAt: "2026-04-14T00:00:01Z",
    });

    act(() => {
      root.render(
        <Harness
          templateKey="tmpl-save"
          blocks={[{ id: "p-1", type: "paragraph", content: "abc" }]}
        />,
      );
    });
    await flush();

    const saveButton = container.querySelector('[data-testid="save"]') as HTMLButtonElement | null;
    expect(saveButton).toBeTruthy();

    await act(async () => {
      saveButton?.click();
      await Promise.resolve();
    });

    expect(saveDraft).toHaveBeenCalledWith("tmpl-save", {
      blocks: [{ id: "p-1", type: "paragraph", content: "abc" }],
      meta: { page: { marginLeftMm: 22, marginRightMm: 20 } },
      lockVersion: 3,
    });
  });

  it("uses meta updated via updateDraftMeta in subsequent saveDraft call", async () => {
    const { getTemplate, saveDraft } = await import("../../../api/templates");

    vi.mocked(getTemplate).mockResolvedValueOnce({
      templateKey: "tmpl-save-updated-meta",
      profileCode: "po",
      name: "Draft template",
      status: "draft",
      lockVersion: 5,
      hasStrippedFields: false,
      blocks: [],
      meta: { page: { marginTopMm: 25, marginRightMm: 20, marginBottomMm: 25, marginLeftMm: 25 } },
      updatedAt: "2026-04-14T00:00:00Z",
    });
    vi.mocked(saveDraft).mockResolvedValueOnce({
      templateKey: "tmpl-save-updated-meta",
      profileCode: "po",
      name: "Draft template",
      status: "draft",
      lockVersion: 6,
      hasStrippedFields: false,
      blocks: [{ id: "p-2", type: "paragraph", content: "xyz" }],
      meta: { page: { marginTopMm: 31, marginRightMm: 22, marginBottomMm: 33, marginLeftMm: 24 } },
      updatedAt: "2026-04-14T00:00:01Z",
    });

    act(() => {
      root.render(
        <Harness
          templateKey="tmpl-save-updated-meta"
          blocks={[{ id: "p-2", type: "paragraph", content: "xyz" }]}
        />,
      );
    });
    await flush();

    const updateMetaButton = container.querySelector('[data-testid="update-meta"]') as HTMLButtonElement | null;
    const saveButton = container.querySelector('[data-testid="save"]') as HTMLButtonElement | null;
    expect(updateMetaButton).toBeTruthy();
    expect(saveButton).toBeTruthy();

    await act(async () => {
      updateMetaButton?.click();
      await Promise.resolve();
    });

    await act(async () => {
      saveButton?.click();
      await Promise.resolve();
    });

    expect(saveDraft).toHaveBeenCalledWith("tmpl-save-updated-meta", {
      blocks: [{ id: "p-2", type: "paragraph", content: "xyz" }],
      meta: { page: { marginTopMm: 31, marginRightMm: 22, marginBottomMm: 33, marginLeftMm: 24 } },
      lockVersion: 5,
    });
  });

  it("replaces the local draft when server-side acknowledgement returns updated state", async () => {
    const { getTemplate } = await import("../../../api/templates");

    vi.mocked(getTemplate).mockResolvedValueOnce({
      templateKey: "tmpl-stripped",
      profileCode: "po",
      name: "Imported template",
      status: "draft",
      lockVersion: 4,
      hasStrippedFields: true,
      blocks: [],
      updatedAt: "2026-04-14T00:00:00Z",
    });

    act(() => {
      root.render(
        <Harness
          templateKey="tmpl-stripped"
          blocks={[]}
          replacementDraft={{
            templateKey: "tmpl-stripped",
            profileCode: "po",
            name: "Imported template",
            status: "draft",
            lockVersion: 5,
            hasStrippedFields: false,
            blocks: [],
            updatedAt: "2026-04-14T00:00:01Z",
          }}
        />,
      );
    });
    await flush();

    const replaceButton = container.querySelector('[data-testid="replace-draft"]') as HTMLButtonElement | null;
    expect(replaceButton).toBeTruthy();

    await act(async () => {
      replaceButton?.click();
      await Promise.resolve();
    });

    expect(container.querySelector('[data-testid="lock-version"]')?.textContent).toBe("5");
    expect(container.querySelector('[data-testid="has-stripped-fields"]')?.textContent).toBe("false");
  });
});
