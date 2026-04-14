import { useState } from "react";
import { PALETTE_BLOCKS, canInsertBlock } from "./block-palette-rules";
import type { BlockCategory } from "./block-palette-rules";
import { useTemplatesStore } from "../../store/templates.store";

interface BlockPaletteProps {
  editor: any | null;
}

const CATEGORY_LABELS: Record<BlockCategory, string> = {
  layout: "Layout",
  field: "Campos",
  data: "Dados",
};

const CATEGORIES: BlockCategory[] = ["layout", "field", "data"];

export function BlockPalette({ editor }: BlockPaletteProps) {
  const [collapsed, setCollapsed] = useState(false);
  const [insertError, setInsertError] = useState<string | null>(null);
  const selectedBlockId = useTemplatesStore((state) => state.selectedBlockId);

  function getCursorBlock(): any | null {
    if (!editor) return null;
    try {
      const pos = editor.getTextCursorPosition?.();
      return pos?.block ?? null;
    } catch {
      return null;
    }
  }

  function findBlockById(blocks: any[], blockId: string): any | null {
    for (const block of blocks) {
      if (block.id === blockId) return block;
      if (Array.isArray(block.children) && block.children.length > 0) {
        const found = findBlockById(block.children, blockId);
        if (found) return found;
      }
    }
    return null;
  }

  function findContainingSectionId(
    blocks: any[],
    targetId: string,
    currentSectionId: string | null = null,
  ): string | null | undefined {
    for (const block of blocks) {
      const nextSectionId = block.type === "section" ? String(block.id) : currentSectionId;
      if (block.id === targetId) return nextSectionId;
      if (Array.isArray(block.children) && block.children.length > 0) {
        const found = findContainingSectionId(block.children, targetId, nextSectionId);
        if (found !== undefined) return found;
      }
    }
    return undefined;
  }

  function getParentBlockType(targetId: string): string | null {
    const result = (function findParentType(
      blocks: any[],
      targetId: string,
      parentType: string | null,
    ): string | null | undefined {
      for (const block of blocks) {
        if (block.id === targetId) return parentType;
        if (Array.isArray(block.children) && block.children.length > 0) {
          const found = findParentType(block.children, targetId, block.type as string);
          if (found !== undefined) return found;
        }
      }
      return undefined;
    })(editor.document ?? [], targetId, null);

    return result ?? null;
  }

  function ensureRootSectionId(): string {
    const doc = (editor.document as any[]) ?? [];
    const existingRootSection = doc.find((block) => block.type === "section");
    if (existingRootSection?.id) return String(existingRootSection.id);

    const sectionBlock = {
      type: "section",
      props: { title: "Nova secao" },
      children: [],
    };

    if (doc.length === 0 && typeof editor.replaceBlocks === "function") {
      const result = editor.replaceBlocks([], [sectionBlock]);
      const inserted = result?.insertedBlocks?.[0];
      if (inserted?.id) return String(inserted.id);
      const fallback = (editor.document as any[])?.find((block) => block.type === "section");
      if (fallback?.id) return String(fallback.id);
      throw new Error("Falha ao criar secao raiz.");
    }

    const referenceBlock = getCursorBlock() ?? doc[doc.length - 1];
    if (!referenceBlock) {
      throw new Error("Falha ao localizar bloco de referencia para criar secao.");
    }
    const inserted = editor.insertBlocks([sectionBlock], referenceBlock, "after");
    if (inserted?.[0]?.id) return String(inserted[0].id);

    const fallback = (editor.document as any[])?.find((block) => block.type === "section");
    if (fallback?.id) return String(fallback.id);
    throw new Error("Falha ao inserir secao raiz.");
  }

  function insertSectionAtRoot(sectionBlock: Record<string, unknown>) {
    const doc = (editor.document as any[]) ?? [];

    if (doc.length === 0 && typeof editor.replaceBlocks === "function") {
      editor.replaceBlocks([], [sectionBlock]);
      return;
    }

    const cursorBlock = getCursorBlock();
    if (!cursorBlock) {
      editor.insertBlocks([sectionBlock], doc[doc.length - 1], "after");
      return;
    }

    const sectionId = findContainingSectionId(doc, String(cursorBlock.id)) ?? null;
    const sectionRef = sectionId ? findBlockById(doc, sectionId) : null;
    const rootReference = sectionRef ?? cursorBlock;

    editor.insertBlocks([sectionBlock], rootReference, "after");
  }

  function insertChildInSection(sectionId: string, blockToInsert: Record<string, unknown>) {
    const section = editor.getBlock?.(sectionId) ?? findBlockById(editor.document ?? [], sectionId);
    if (!section || section.type !== "section") {
      throw new Error("Secao de destino nao encontrada.");
    }

    const children = Array.isArray(section.children) ? section.children : [];
    editor.updateBlock(sectionId, {
      children: [...children, blockToInsert],
    });
  }

  function buildBlockToInsert(type: string, defaultProps: Record<string, unknown>) {
    if (type === "dataTable") {
      return {
        type,
        props: defaultProps,
        content: {
          type: "tableContent",
          headerRows: 0,
          rows: [
            {
              cells: [[{ type: "text", text: "" }]],
            },
          ],
        },
      };
    }

    return { type, props: defaultProps, children: [] };
  }

  function handleInsert(type: string, defaultProps: Record<string, unknown>) {
    setInsertError(null);

    if (!editor) {
      setInsertError("Editor nao esta pronto.");
      return;
    }

    const cursorBlock = getCursorBlock();
    const selectedBlock =
      selectedBlockId && typeof editor.getBlock === "function"
        ? editor.getBlock(selectedBlockId)
        : null;
    const activeBlock = selectedBlock ?? cursorBlock;
    const activeBlockId = activeBlock?.id ? String(activeBlock.id) : null;
    const parentBlockType = activeBlockId ? getParentBlockType(activeBlockId) : null;
    const currentBlockType = typeof activeBlock?.type === "string" ? activeBlock.type : null;

    const error = canInsertBlock(type, parentBlockType, currentBlockType);
    if (error) {
      setInsertError(error);
      return;
    }

    try {
      const blockToInsert = buildBlockToInsert(type, defaultProps);

      if (type === "section") {
        insertSectionAtRoot(blockToInsert);
        return;
      }

      const doc = (editor.document as any[]) ?? [];
      const sectionIdFromContext =
        currentBlockType === "section" && activeBlockId
          ? activeBlockId
          : activeBlockId
            ? (findContainingSectionId(doc, activeBlockId) ?? null)
            : null;

      const sectionId = sectionIdFromContext ?? ensureRootSectionId();
      insertChildInSection(sectionId, blockToInsert);
    } catch (err) {
      setInsertError(err instanceof Error ? err.message : "Erro ao inserir bloco.");
    }
  }

  if (collapsed) {
    return (
      <div
        data-testid="block-palette"
        data-contrast="high"
        style={{
          width: "32px",
          flexShrink: 0,
          background: "#20222c",
          borderRight: "1px solid rgba(255,255,255,0.08)",
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          paddingTop: "8px",
        }}
      >
        <button title="Expandir paleta" onClick={() => setCollapsed(false)} style={toggleBtnStyle}>
          {">"}
        </button>
      </div>
    );
  }

  return (
    <div
      data-testid="block-palette"
      data-contrast="high"
      style={{
        width: "220px",
        flexShrink: 0,
        background: "#20222c",
        color: "rgba(255,255,255,0.92)",
        borderRight: "1px solid rgba(255,255,255,0.08)",
        display: "flex",
        flexDirection: "column",
        overflow: "hidden",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          padding: "8px 10px",
          borderBottom: "1px solid rgba(255,255,255,0.08)",
          flexShrink: 0,
        }}
      >
        <span
          style={{
            fontSize: "11px",
            fontWeight: 600,
            color: "rgba(255,255,255,0.55)",
            textTransform: "uppercase",
            letterSpacing: "0.06em",
          }}
        >
          Blocos
        </span>
        <button title="Colapsar paleta" onClick={() => setCollapsed(true)} style={toggleBtnStyle}>
          {"<"}
        </button>
      </div>

      {insertError && (
        <div
          role="alert"
          style={{
            margin: "8px",
            padding: "6px 8px",
            background: "rgba(239,68,68,0.12)",
            border: "1px solid rgba(239,68,68,0.3)",
            borderRadius: "4px",
            fontSize: "11px",
            color: "#fca5a5",
            flexShrink: 0,
          }}
        >
          {insertError}
          <button
            onClick={() => setInsertError(null)}
            style={{
              float: "right",
              background: "none",
              border: "none",
              color: "#fca5a5",
              cursor: "pointer",
              padding: "0",
              lineHeight: 1,
            }}
            aria-label="Fechar erro"
          >
            x
          </button>
        </div>
      )}

      <div style={{ overflowY: "auto", flex: 1 }}>
        {CATEGORIES.map((category) => {
          const blocks = PALETTE_BLOCKS.filter((block) => block.category === category);
          return (
            <div key={category} style={{ paddingBottom: "4px" }}>
              <div
                style={{
                  padding: "6px 10px 3px",
                  fontSize: "10px",
                  fontWeight: 600,
                  color: "rgba(255,255,255,0.55)",
                  textTransform: "uppercase",
                  letterSpacing: "0.07em",
                }}
              >
                {CATEGORY_LABELS[category]}
              </div>
              {blocks.map((block) => (
                <button
                  key={block.type}
                  data-testid={`palette-insert-${block.type}`}
                  onClick={() => handleInsert(block.type, block.defaultProps)}
                  style={paletteItemStyle}
                  onMouseEnter={(e) => {
                    (e.currentTarget as HTMLButtonElement).style.background = "rgba(255,255,255,0.07)";
                  }}
                  onMouseLeave={(e) => {
                    (e.currentTarget as HTMLButtonElement).style.background = "transparent";
                  }}
                >
                  <span style={{ fontSize: "13px", lineHeight: 1 }}>{blockIcon(block.type)}</span>
                  <span style={{ fontSize: "12px", color: "rgba(255,255,255,0.92)" }}>{block.label}</span>
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
  background: "none",
  border: "none",
  color: "rgba(255,255,255,0.78)",
  cursor: "pointer",
  fontSize: "16px",
  lineHeight: 1,
  padding: "2px 4px",
  borderRadius: "3px",
};

const paletteItemStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: "8px",
  width: "100%",
  padding: "5px 10px",
  background: "transparent",
  border: "none",
  cursor: "pointer",
  textAlign: "left",
  transition: "background 0.1s",
};

function blockIcon(type: string): string {
  switch (type) {
    case "section":
      return "S";
    case "field":
      return "F";
    case "dataTable":
      return "T";
    case "repeatable":
      return "R";
    case "richBlock":
      return "B";
    default:
      return "?";
  }
}
