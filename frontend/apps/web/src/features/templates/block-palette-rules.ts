// Block insertion rules for the template editor palette.
// Sections can only be inserted at the top level.
// Other block types can only be inserted inside a section's children.

export type BlockCategory = 'layout' | 'field' | 'data';

export interface PaletteBlock {
  type: string;
  label: string;     // PT-BR
  category: BlockCategory;
  defaultProps: Record<string, unknown>;
  topLevelOnly: boolean; // if true, can only insert at root level
}

export const PALETTE_BLOCKS: PaletteBlock[] = [
  {
    type: 'section',
    label: 'Seção',
    category: 'layout',
    topLevelOnly: true,
    defaultProps: { title: 'Nova seção' },
  },
  {
    type: 'field',
    label: 'Campo',
    category: 'field',
    topLevelOnly: false,
    defaultProps: { label: 'Novo campo', fieldType: 'text' },
  },
  {
    type: 'dataTable',
    label: 'Tabela de dados',
    category: 'data',
    topLevelOnly: false,
    defaultProps: { columns: [], minRows: 1, maxRows: 10 },
  },
  {
    type: 'repeatable',
    label: 'Repetível',
    category: 'layout',
    topLevelOnly: false,
    defaultProps: { minItems: 1, maxItems: 10 },
  },
  {
    type: 'richBlock',
    label: 'Bloco rico',
    category: 'field',
    topLevelOnly: false,
    defaultProps: { label: 'Bloco rico' },
  },
];

// Validate that a block can be inserted at the given insertion context.
// Returns null if valid, or an error string if not.
export function canInsertBlock(blockType: string, parentBlockType: string | null): string | null {
  const rule = PALETTE_BLOCKS.find(b => b.type === blockType);
  if (!rule) return `Unknown block type: ${blockType}`;
  if (rule.topLevelOnly && parentBlockType !== null) {
    return `Seções só podem ser inseridas no nível raiz.`;
  }
  if (!rule.topLevelOnly && parentBlockType === null) {
    return `Este bloco só pode ser inserido dentro de uma seção.`;
  }
  return null;
}
