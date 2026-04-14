// Block insertion rules for the template editor palette.
// Sections are always top-level.
// Other block types must be inserted inside a section context.

export type BlockCategory = "layout" | "field" | "data";

export interface PaletteBlock {
  type: string;
  label: string;
  category: BlockCategory;
  defaultProps: Record<string, unknown>;
  topLevelOnly: boolean;
}

export const PALETTE_BLOCKS: PaletteBlock[] = [
  {
    type: "section",
    label: "Secao",
    category: "layout",
    topLevelOnly: true,
    defaultProps: { title: "Nova secao" },
  },
  {
    type: "dataTable",
    label: "Tabela de dados",
    category: "data",
    topLevelOnly: false,
    defaultProps: { columns: [], minRows: 1, maxRows: 10 },
  },
  {
    type: "repeatable",
    label: "Repetivel",
    category: "layout",
    topLevelOnly: false,
    defaultProps: { minItems: 1, maxItems: 10 },
  },
  {
    type: "richBlock",
    label: "Bloco rico",
    category: "field",
    topLevelOnly: false,
    defaultProps: { label: "Bloco rico" },
  },
];

// Returns null if valid, or an error string if invalid.
// parentBlockType = direct parent of cursor block.
// currentBlockType = block currently selected by the cursor.
export function canInsertBlock(
  blockType: string,
  parentBlockType: string | null,
  currentBlockType: string | null = null,
): string | null {
  const rule = PALETTE_BLOCKS.find((block) => block.type === blockType);
  if (!rule) return `Unknown block type: ${blockType}`;

  if (rule.topLevelOnly) {
    // Sections are inserted at root by BlockPalette insertion logic.
    return null;
  }

  const hasSectionContext =
    parentBlockType === "section" || currentBlockType === "section";

  if (!hasSectionContext) {
    return "Este bloco so pode ser inserido dentro de uma secao.";
  }

  return null;
}
