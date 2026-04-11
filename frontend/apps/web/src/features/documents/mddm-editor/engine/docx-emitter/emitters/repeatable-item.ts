import {
  Table,
  TableRow,
  TableCell,
  Paragraph,
  TextRun,
  WidthType,
  BorderStyle,
} from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { ptToHalfPt } from "../../helpers/units";

function hexToFill(hex: string): string {
  return hex.replace(/^#/, "").toUpperCase();
}

function isMDDMBlock(child: unknown): child is MDDMBlock {
  return typeof child === "object" && child !== null && typeof (child as MDDMBlock).type === "string" && !("text" in (child as Record<string, unknown>));
}

/** ChildRenderer is supplied by the main emitter so repeatable-item can recursively
 *  emit any block type without depending on the registry directly (avoids cycles). */
export type ChildRenderer = (child: MDDMBlock) => unknown[];

export function emitRepeatableItem(
  block: MDDMBlock,
  tokens: LayoutTokens,
  renderChild: ChildRenderer,
): Table[] {
  const accent = hexToFill(tokens.theme.accent);
  const borderColor = hexToFill(tokens.theme.accentBorder);
  const title = (block.props as { title?: string }).title ?? "";

  const innerChildren: unknown[] = [];
  if (title) {
    innerChildren.push(
      new Paragraph({
        children: [
          new TextRun({
            text: title,
            bold: true,
            size: ptToHalfPt(tokens.typography.baseSizePt),
            font: tokens.typography.exportFont,
          }),
        ],
      }),
    );
  }
  const allChildren = (block.children ?? []) as unknown[];
  for (const child of allChildren) {
    if (isMDDMBlock(child)) {
      innerChildren.push(...renderChild(child));
    }
  }

  const cell = new TableCell({
    borders: {
      top:    { style: BorderStyle.SINGLE, size: 4, color: borderColor },
      bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
      left:   { style: BorderStyle.SINGLE, size: 12, color: accent },
      right:  { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    },
    children: innerChildren as any,
  });

  return [
    new Table({
      width: { size: 100, type: WidthType.PERCENTAGE },
      rows: [new TableRow({ children: [cell] })],
    }),
  ];
}
