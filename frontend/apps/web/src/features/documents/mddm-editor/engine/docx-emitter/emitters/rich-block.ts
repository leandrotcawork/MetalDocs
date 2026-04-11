import { Paragraph, TextRun } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { ptToHalfPt } from "../../helpers/units";
import type { ChildRenderer } from "./repeatable-item";
import { isMDDMBlock } from "../guards";

export function emitRichBlock(
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
            size: ptToHalfPt(tokens.typography.labelSizePt),
            font: tokens.typography.exportFont,
          }),
        ],
      }),
    );
  }

  for (const child of (block.children ?? []) as unknown[]) {
    if (isMDDMBlock(child)) {
      out.push(...renderChild(child));
    }
  }

  return out;
}
