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
import { hexToFill } from "../../helpers/color";
import { interpretRepeatableItem } from "../../layout-interpreter";
import { isMDDMBlock } from "../guards";

/** ChildRenderer is supplied by the main emitter so repeatable-item can recursively
 *  emit any block type without depending on the registry directly (avoids cycles). */
export type ChildRenderer = (child: MDDMBlock) => unknown[];

export function emitRepeatableItem(
  block: MDDMBlock,
  tokens: LayoutTokens,
  renderChild: ChildRenderer,
  itemIndex = 0,
): Table[] {
  const vm = interpretRepeatableItem(
    { props: block.props as Record<string, unknown> },
    tokens,
    { itemIndex },
  );
  const accent = hexToFill(vm.accentBorderColor);
  const borderColor = hexToFill(tokens.theme.accentBorder);

  const displayTitle = vm.title ? `${vm.number} ${vm.title}` : `Item ${vm.number}`;

  const innerChildren: unknown[] = [];
  if (displayTitle) {
    innerChildren.push(
      new Paragraph({
        children: [
          new TextRun({
            text: displayTitle,
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
