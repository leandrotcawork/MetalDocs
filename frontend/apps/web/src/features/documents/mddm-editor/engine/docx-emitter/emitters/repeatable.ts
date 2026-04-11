import { Paragraph, TextRun } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { ptToHalfPt } from "../../helpers/units";
import { emitRepeatableItem, type ChildRenderer } from "./repeatable-item";

function isItemBlock(child: unknown): child is MDDMBlock {
  return typeof child === "object" && child !== null && (child as MDDMBlock).type === "repeatableItem";
}

export function emitRepeatable(
  block: MDDMBlock,
  tokens: LayoutTokens,
  renderChild: ChildRenderer,
): unknown[] {
  const label = (block.props as { label?: string }).label ?? "";
  const out: unknown[] = [];

  if (label) {
    out.push(
      new Paragraph({
        children: [
          new TextRun({
            text: label,
            bold: true,
            size: ptToHalfPt(tokens.typography.baseSizePt),
            font: tokens.typography.exportFont,
          }),
        ],
      }),
    );
  }

  const items = ((block.children ?? []) as unknown[]).filter(isItemBlock) as MDDMBlock[];
  for (const item of items) {
    out.push(...emitRepeatableItem(item, tokens, renderChild));
  }

  return out;
}
