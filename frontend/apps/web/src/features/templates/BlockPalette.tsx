import { useState } from "react";
import { PALETTE_BLOCKS, canInsertBlock } from "./block-palette-rules";
import type { BlockCategory } from "./block-palette-rules";

interface BlockPaletteProps {
  editor: any | null;
}

const CATEGORY_LABELS: Record<BlockCategory, string> = {
  layout: 'Layout',
  field: 'Campos',
  data: 'Dados',
};

const CATEGORIES: BlockCategory[] = ['layout', 'field', 'data'];

export function BlockPalette({ editor }: BlockPaletteProps) {
  const [collapsed, setCollapsed] = useState(false);
  const [insertError, setInsertError] = useState<string | null>(null);

  function getParentBlockType(): string | null {
    if (!editor) return null;
    try {
      const pos = editor.getTextCursorPosition?.();
      if (!pos?.block) return null;

      // Walk up through the editor document to find the parent block type
      const findParent = (
        blocks: any[],
        targetId: string,
        parentType: string | null,
      ): string | null | undefined => {
        for (const block of blocks) {
          if (block.id === targetId) return parentType;
          if (Array.isArray(block.children) && block.children.length > 0) {
            const found = findParent(block.children, targetId, block.type as string);
            if (found !== undefined) return found;
          }
        }
        return undefined;
      };

      const result = findParent(editor.document, pos.block.id, null);
      // undefined = not found (shouldn't happen), null = top-level
      return result ?? null;
    } catch {
      return null;
    }
  }

  function handleInsert(type: string, defaultProps: Record<string, unknown>) {
    setInsertError(null);

    const parentBlockType = getParentBlockType();
    const error = canInsertBlock(type, parentBlockType);
    if (error) {
      setInsertError(error);
      return;
    }

    if (!editor) {
      setInsertError('Editor não está pronto.');
      return;
    }

    try {
      const pos = editor.getTextCursorPosition?.();
      const referenceBlock = pos?.block ?? editor.document[editor.document.length - 1];

      editor.insertBlocks(
        [{ type, props: defaultProps, children: [] }],
        referenceBlock,
        'after',
      );
    } catch (err) {
      setInsertError(err instanceof Error ? err.message : 'Erro ao inserir bloco.');
    }
  }

  if (collapsed) {
    return (
      <div
        style={{
          width: '32px',
          flexShrink: 0,
          background: 'var(--color-surface-2, #1e2028)',
          borderRight: '1px solid var(--color-border, rgba(255,255,255,0.08))',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          paddingTop: '8px',
        }}
      >
        <button
          title="Expandir paleta"
          onClick={() => setCollapsed(false)}
          style={toggleBtnStyle}
        >
          {'›'}
        </button>
      </div>
    );
  }

  return (
    <div
      data-testid="block-palette"
      style={{
        width: '200px',
        flexShrink: 0,
        background: 'var(--color-surface-2, #1e2028)',
        borderRight: '1px solid var(--color-border, rgba(255,255,255,0.08))',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
      }}
    >
      {/* Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '8px 10px',
          borderBottom: '1px solid var(--color-border, rgba(255,255,255,0.08))',
          flexShrink: 0,
        }}
      >
        <span style={{ fontSize: '11px', fontWeight: 600, color: 'rgba(255,255,255,0.5)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>
          Blocos
        </span>
        <button
          title="Colapsar paleta"
          onClick={() => setCollapsed(true)}
          style={toggleBtnStyle}
        >
          {'‹'}
        </button>
      </div>

      {/* Error message */}
      {insertError && (
        <div
          role="alert"
          style={{
            margin: '8px',
            padding: '6px 8px',
            background: 'rgba(239,68,68,0.12)',
            border: '1px solid rgba(239,68,68,0.3)',
            borderRadius: '4px',
            fontSize: '11px',
            color: '#fca5a5',
            flexShrink: 0,
          }}
        >
          {insertError}
          <button
            onClick={() => setInsertError(null)}
            style={{ float: 'right', background: 'none', border: 'none', color: '#fca5a5', cursor: 'pointer', padding: '0', lineHeight: 1 }}
            aria-label="Fechar erro"
          >
            ×
          </button>
        </div>
      )}

      {/* Block groups */}
      <div style={{ overflowY: 'auto', flex: 1 }}>
        {CATEGORIES.map(category => {
          const blocks = PALETTE_BLOCKS.filter(b => b.category === category);
          return (
            <div key={category} style={{ paddingBottom: '4px' }}>
              <div
                style={{
                  padding: '6px 10px 3px',
                  fontSize: '10px',
                  fontWeight: 600,
                  color: 'rgba(255,255,255,0.3)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.07em',
                }}
              >
                {CATEGORY_LABELS[category]}
              </div>
              {blocks.map(block => (
                <button
                  key={block.type}
                  data-testid={`palette-insert-${block.type}`}
                  onClick={() => handleInsert(block.type, block.defaultProps)}
                  style={paletteItemStyle}
                  onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.background = 'rgba(255,255,255,0.07)'; }}
                  onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = 'transparent'; }}
                >
                  <span style={{ fontSize: '13px', lineHeight: 1 }}>{blockIcon(block.type)}</span>
                  <span style={{ fontSize: '12px', color: 'rgba(255,255,255,0.75)' }}>{block.label}</span>
                </button>
              ))}
            </div>
          );
        })}
      </div>
    </div>
  );
}

const toggleBtnStyle: React.CSSProperties = {
  background: 'none',
  border: 'none',
  color: 'rgba(255,255,255,0.4)',
  cursor: 'pointer',
  fontSize: '16px',
  lineHeight: 1,
  padding: '2px 4px',
  borderRadius: '3px',
};

const paletteItemStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: '8px',
  width: '100%',
  padding: '5px 10px',
  background: 'transparent',
  border: 'none',
  cursor: 'pointer',
  textAlign: 'left',
  transition: 'background 0.1s',
};

function blockIcon(type: string): string {
  switch (type) {
    case 'section': return '▦';
    case 'field': return '⊞';
    case 'dataTable': return '⊟';
    case 'repeatable': return '↻';
    case 'richBlock': return '✎';
    default: return '□';
  }
}
