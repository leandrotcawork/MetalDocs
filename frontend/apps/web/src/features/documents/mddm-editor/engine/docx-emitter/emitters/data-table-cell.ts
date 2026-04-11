import { TableCell, Paragraph, TextRun, BorderStyle } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { mddmTextRunsToDocxRuns } from "../inline-content";
import { extractTextRuns } from "./paragraph";
import { hexToFill } from "../../helpers/color";

export function emitDataTableCell(block: MDDMBlock, tokens: LayoutTokens): TableCell {
  const borderColor = hexToFill(tokens.theme.accentBorder);
  const borders = {
    top:    { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    left:   { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    right:  { style: BorderStyle.SINGLE, size: 4, color: borderColor },
  };

  const runs = mddmTextRunsToDocxRuns(extractTextRuns(block), tokens);

  return new TableCell({
    borders,
    children: [
      new Paragraph({
        children: runs.length > 0 ? runs : [new TextRun({ text: "" })],
      }),
    ],
  });
}
