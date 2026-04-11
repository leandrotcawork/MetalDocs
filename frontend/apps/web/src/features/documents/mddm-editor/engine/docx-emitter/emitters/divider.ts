import { Paragraph, BorderStyle } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

export function emitDivider(_block: MDDMBlock, tokens: LayoutTokens): Paragraph[] {
  const color = tokens.theme.accentBorder.replace(/^#/, "").toUpperCase();
  const options = {
    border: { bottom: { style: BorderStyle.SINGLE, size: 6, color, space: 1 } },
    children: [] as const,
  } as const;
  const paragraph = new Paragraph(options);
  (paragraph as unknown as { options: typeof options }).options = options;
  return [paragraph];
}
