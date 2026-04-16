import { Paragraph, TextRun } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../shared/adapter";
import { ptToHalfPt } from "../../shared/helpers/units";
import { interpretRichBlock } from "../../layout-interpreter";
import type { ChildRenderer } from "./repeatable-item";
import { isMDDMBlock } from "../guards";

export function emitRichBlock(
  block: MDDMBlock,
  tokens: LayoutTokens,
  renderChild: ChildRenderer,
): unknown[] {
  const vm = interpretRichBlock(
    { props: block.props as Record<string, unknown> },
    tokens,
  );
  const fontSizePt = parseFloat(vm.labelFontSize); // e.g. "9pt" â†’ 9
  const out: unknown[] = [];

  if (vm.label) {
    out.push(
      new Paragraph({
        children: [
          new TextRun({
            text: vm.label,
            bold: true,
            size: ptToHalfPt(isNaN(fontSizePt) ? tokens.typography.labelSizePt : fontSizePt),
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

