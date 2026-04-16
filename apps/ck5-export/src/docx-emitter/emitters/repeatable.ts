import { Paragraph, TextRun } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../shared/adapter";
import { ptToHalfPt } from "../../shared/helpers/units";
import { interpretRepeatable } from "../../layout-interpreter";
import { emitRepeatableItem, type ChildRenderer } from "./repeatable-item";

function isItemBlock(child: unknown): child is MDDMBlock {
  return typeof child === "object" && child !== null && (child as MDDMBlock).type === "repeatableItem";
}

export function emitRepeatable(
  block: MDDMBlock,
  tokens: LayoutTokens,
  renderChild: ChildRenderer,
): unknown[] {
  const vm = interpretRepeatable(
    { props: block.props as Record<string, unknown>, children: block.children as unknown[] },
    tokens,
  );
  const out: unknown[] = [];

  if (vm.label) {
    out.push(
      new Paragraph({
        children: [
          new TextRun({
            text: vm.label,
            bold: true,
            size: ptToHalfPt(tokens.typography.baseSizePt),
            font: tokens.typography.exportFont,
          }),
        ],
      }),
    );
  }

  const items = ((block.children ?? []) as unknown[]).filter(isItemBlock) as MDDMBlock[];
  for (let i = 0; i < items.length; i++) {
    out.push(...emitRepeatableItem(items[i], tokens, renderChild, i));
  }

  return out;
}

