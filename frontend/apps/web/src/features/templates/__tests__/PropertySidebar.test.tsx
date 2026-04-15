import { describe, it, expect, vi, afterEach } from "vitest";
import React from "react";
import { createRoot } from "react-dom/client";
import { act } from "react";
import { PropertySidebar } from "../PropertySidebar";
import { defaultTemplatePageSettings } from "../page-settings";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

function setup() {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
}

function renderSidebar(props: React.ComponentProps<typeof PropertySidebar>) {
  act(() => {
    root.render(<PropertySidebar {...props} />);
  });
}

function makeSidebarProps(overrides: Partial<React.ComponentProps<typeof PropertySidebar>> = {}): React.ComponentProps<typeof PropertySidebar> {
  return {
    editor: makeEditor(null),
    selectedBlockId: null,
    pageSettings: defaultTemplatePageSettings,
    onPageSettingsChange: vi.fn(),
    ...overrides,
  };
}

afterEach(() => {
  act(() => {
    root.unmount();
  });
  container.remove();
});

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

function makeEditor(block: Record<string, unknown> | null) {
  return {
    getBlock: vi.fn((id: string) => {
      if (block && block.id === id) return block;
      return undefined;
    }),
    updateBlock: vi.fn(),
  };
}

const SECTION_BLOCK = {
  id: "block-1",
  type: "section",
  props: {
    title: "Dados do cliente",
    styleJson: JSON.stringify({ headerBackground: "#ff0000", headerColor: "#ffffff" }),
    capabilitiesJson: JSON.stringify({ locked: true, removable: false, reorderable: false }),
  },
};

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("PropertySidebar", () => {
  it("shows page margin controls when no block is selected", () => {
    setup();
    const editor = makeEditor(null);
    renderSidebar(makeSidebarProps({ editor }));

    expect(container.textContent).toContain("Margem superior (mm)");
    expect(container.textContent).toContain("Margem direita (mm)");
    expect(container.textContent).toContain("Margem inferior (mm)");
    expect(container.textContent).toContain("Margem esquerda (mm)");
  });

  it("calls onPageSettingsChange when page margin control changes", () => {
    setup();
    const editor = makeEditor(SECTION_BLOCK);
    const onPageSettingsChange = vi.fn();
    renderSidebar(makeSidebarProps({
      editor,
      selectedBlockId: "block-1",
      pageSettings: defaultTemplatePageSettings,
      onPageSettingsChange,
    }));

    const marginTopInput = container.querySelector(
      '[data-testid="page-margin-top-mm"]'
    ) as HTMLInputElement | null;

    expect(marginTopInput).toBeTruthy();

    act(() => {
      if (!marginTopInput) return;
      const nativeInputValueSetter = Object.getOwnPropertyDescriptor(
        window.HTMLInputElement.prototype,
        "value"
      )?.set;
      if (nativeInputValueSetter) {
        nativeInputValueSetter.call(marginTopInput, "30");
      } else {
        marginTopInput.value = "30";
      }
      marginTopInput.dispatchEvent(new Event("input", { bubbles: true }));
      marginTopInput.dispatchEvent(new Event("change", { bubbles: true }));
    });

    expect(onPageSettingsChange).toHaveBeenCalledWith({
      marginTopMm: 30,
      marginRightMm: defaultTemplatePageSettings.marginRightMm,
      marginBottomMm: defaultTemplatePageSettings.marginBottomMm,
      marginLeftMm: defaultTemplatePageSettings.marginLeftMm,
    });
  });

  it("1. shows placeholder when no block is selected", () => {
    setup();
    const editor = makeEditor(null);
    renderSidebar(makeSidebarProps({ editor, selectedBlockId: null }));

    expect(container.textContent).toContain("Selecione um bloco para editar suas propriedades");
  });

  it("2. Propriedades tab shows title input when section block is selected", () => {
    setup();
    const editor = makeEditor(SECTION_BLOCK);
    renderSidebar(makeSidebarProps({ editor, selectedBlockId: "block-1" }));

    // Default tab is Propriedades
    const inputs = container.querySelectorAll('input[type="text"]');
    const titleInput = Array.from(inputs).find(
      (el) => (el as HTMLInputElement).value === "Dados do cliente"
    ) as HTMLInputElement | undefined;

    expect(titleInput).toBeTruthy();
    expect(titleInput!.value).toBe("Dados do cliente");
  });

  it("3. Estilo tab renders color controls from sectionStyleFieldSchema", () => {
    setup();
    const editor = makeEditor(SECTION_BLOCK);
    renderSidebar(makeSidebarProps({ editor, selectedBlockId: "block-1" }));

    // Click Estilo tab
    const estiloBtn = Array.from(container.querySelectorAll("button")).find(
      (b) => b.textContent === "Estilo"
    ) as HTMLButtonElement;
    act(() => { estiloBtn.click(); });

    // Should contain color labels from sectionStyleFieldSchema
    expect(container.textContent).toContain("Fundo do cabeçalho");
    expect(container.textContent).toContain("Cor do texto do cabeçalho");
  });

  it("4. Capacidades tab renders toggle controls from sectionCapsFieldSchema", () => {
    setup();
    const editor = makeEditor(SECTION_BLOCK);
    renderSidebar(makeSidebarProps({ editor, selectedBlockId: "block-1" }));

    // Click Capacidades tab
    const capsBtn = Array.from(container.querySelectorAll("button")).find(
      (b) => b.textContent === "Capacidades"
    ) as HTMLButtonElement;
    act(() => { capsBtn.click(); });

    // Should render toggles for sectionCapsFieldSchema
    expect(container.textContent).toContain("Bloqueado");
    expect(container.textContent).toContain("Removível");
    expect(container.textContent).toContain("Reordenável");

    // Toggles should be checkboxes
    const checkboxes = container.querySelectorAll('input[type="checkbox"]');
    expect(checkboxes.length).toBeGreaterThanOrEqual(3);
  });

  it("5. Changing a style color control calls editor.updateBlock with correct args", () => {
    setup();
    const editor = makeEditor(SECTION_BLOCK);
    renderSidebar(makeSidebarProps({ editor, selectedBlockId: "block-1" }));

    // Switch to Estilo tab
    const estiloBtn = Array.from(container.querySelectorAll("button")).find(
      (b) => b.textContent === "Estilo"
    ) as HTMLButtonElement;
    act(() => { estiloBtn.click(); });

    // Find color inputs
    const colorInputs = container.querySelectorAll('input[type="color"]');
    expect(colorInputs.length).toBeGreaterThanOrEqual(1);

    // Fire change on first color input (headerBackground) using React's native value setter
    // so that React's synthetic onChange handler fires correctly in jsdom.
    act(() => {
      const input = colorInputs[0] as HTMLInputElement;
      const nativeInputValueSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, "value")?.set;
      if (nativeInputValueSetter) {
        nativeInputValueSetter.call(input, "#123456");
      } else {
        input.value = "#123456";
      }
      input.dispatchEvent(new Event("change", { bubbles: true }));
    });

    expect(editor.updateBlock).toHaveBeenCalledWith("block-1", {
      props: {
        styleJson: expect.stringContaining('"headerBackground":"#123456"'),
      },
    });
  });

  it("shows non-configurable message for unknown block type", () => {
    setup();
    const editor = makeEditor({ id: "block-x", type: "unknownType", props: {} });
    renderSidebar(makeSidebarProps({ editor, selectedBlockId: "block-x" }));
    expect(container.textContent).toContain("Tipo de bloco não configurável");
  });

  it("shows readonly message for repeatableItem in Propriedades tab", () => {
    setup();
    const block = { id: "item-1", type: "repeatableItem", props: {} };
    const editor = makeEditor(block);
    renderSidebar(makeSidebarProps({ editor, selectedBlockId: "item-1" }));
    expect(container.textContent).toContain("Número do item determinado pelo bloco pai");
  });
});
