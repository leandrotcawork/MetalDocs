import { Paragraph } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { mddmTextRunsToDocxRuns } from "../inline-content";
import { extractTextRuns } from "./paragraph";

export function emitBulletListItem(block: MDDMBlock, tokens: LayoutTokens): Paragraph[] {
  const runs = mddmTextRunsToDocxRuns(extractTextRuns(block), tokens);
  return [
    new Paragraph({
      bullet: { level: 0 },
      children: runs,
    }),
  ];
}
