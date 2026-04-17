import { BorderStyle, Paragraph, Table, TableCell, TableRow, TextRun, WidthType } from "docx"
import type { ExportNode } from "../export-node"
import type { LayoutTokens } from "../layout-ir"
import { hexToFill } from "./helpers"

export function emitField(node: ExportNode, tokens: LayoutTokens): Table[] {
  if (node.kind !== "field") {
    return []
  }

  const borderColor = hexToFill(tokens.theme.accentBorder)
  const labelText = node.label ?? node.id
  const borders = {
    top: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    left: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    right: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
  }

  return [
    new Table({
      width: { size: 100, type: WidthType.PERCENTAGE },
      rows: [
        new TableRow({
          children: [
            new TableCell({
              width: { size: 35, type: WidthType.PERCENTAGE },
              shading: { fill: hexToFill(tokens.theme.accentLight), type: "clear", color: "auto" },
              borders,
              children: [new Paragraph({ children: [new TextRun({ text: labelText })] })],
            }),
            new TableCell({
              width: { size: 65, type: WidthType.PERCENTAGE },
              borders,
              children: [new Paragraph({ children: [new TextRun({ text: node.value })] })],
            }),
          ],
        }),
      ],
    }),
  ]
}
