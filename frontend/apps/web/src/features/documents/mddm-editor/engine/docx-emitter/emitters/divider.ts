import { Paragraph, BorderStyle } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

export function emitDivider(_block: MDDMBlock, tokens: LayoutTokens): Paragraph[] {
  const color = tokens.theme.accentBorder.replace(/^#/, "").toUpperCase();
  return [
    new Paragraph({
      border: {
        bottom: { style: BorderStyle.SINGLE, size: 6, color, space: 1 },
      },
      children: [],
    }),
  ];
}
