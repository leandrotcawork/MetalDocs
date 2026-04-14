import React from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { TemplateListPanel } from "../TemplateListPanel";

const navigateMock = vi.fn();

vi.mock("react-router-dom", () => ({
  useNavigate: () => navigateMock,
}));

vi.mock("../../../api/templates", async () => {
  const actual = await vi.importActual<typeof import("../../../api/templates")>("../../../api/templates");
  return {
    ...actual,
    listTemplates: vi.fn(),
    createTemplate: vi.fn(),
    editPublished: vi.fn(),
    cloneTemplate: vi.fn(),
    deleteDraft: vi.fn(),
    discardDraft: vi.fn(),
    deprecateTemplate: vi.fn(),
    exportTemplate: vi.fn(),
  };
});

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
});

afterEach(() => {
  act(() => {
    root.unmount();
  });
  container.remove();
  vi.clearAllMocks();
});

describe("TemplateListPanel behaviors", () => {
  it("calls editPublished before navigating when editing a published template", async () => {
    const { listTemplates, editPublished } = await import("../../../api/templates");

    vi.mocked(listTemplates).mockResolvedValueOnce([
      { templateKey: "tmpl-published", version: 4, profileCode: "po", name: "Published", status: "published" },
    ]);
    vi.mocked(editPublished).mockResolvedValueOnce({
      templateKey: "tmpl-published",
      profileCode: "po",
      name: "Published",
      status: "draft",
      lockVersion: 1,
      hasStrippedFields: false,
      blocks: [],
      updatedAt: "2026-04-14T00:00:00Z",
    });

    act(() => {
      root.render(<TemplateListPanel profileCode="po" />);
    });
    await flush();
    await flush();

    const editButton = Array.from(container.querySelectorAll("button")).find((button) => button.textContent === "Editar");
    expect(editButton).toBeTruthy();

    await act(async () => {
      (editButton as HTMLButtonElement).click();
      await Promise.resolve();
    });

    expect(editPublished).toHaveBeenCalledWith("tmpl-published");
    expect(navigateMock).toHaveBeenCalledWith("/registry/profiles/po/templates/tmpl-published/edit");
  });

  it("navigates to the cloned draft after a clone action succeeds", async () => {
    const { listTemplates, cloneTemplate } = await import("../../../api/templates");

    vi.mocked(listTemplates).mockResolvedValueOnce([
      { templateKey: "tmpl-source", version: 2, profileCode: "po", name: "Source", status: "published" },
    ]);
    vi.mocked(cloneTemplate).mockResolvedValueOnce({
      templateKey: "tmpl-clone",
      profileCode: "po",
      name: "Source (copia)",
      status: "draft",
      lockVersion: 1,
      hasStrippedFields: false,
      blocks: [],
      updatedAt: "2026-04-14T00:00:00Z",
    });

    act(() => {
      root.render(<TemplateListPanel profileCode="po" />);
    });
    await flush();
    await flush();

    const cloneButton = Array.from(container.querySelectorAll("button")).find((button) => button.textContent === "Clonar");
    expect(cloneButton).toBeTruthy();

    await act(async () => {
      (cloneButton as HTMLButtonElement).click();
      await Promise.resolve();
    });

    expect(cloneTemplate).toHaveBeenCalledWith("tmpl-source", "Source (copia)");
    expect(navigateMock).toHaveBeenCalledWith("/registry/profiles/po/templates/tmpl-clone/edit");
  });
});
