import { Paragraph } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../shared/adapter";
import { mddmTextRunsToDocxRuns } from "../inline-content";
import { extractTextRuns } from "./paragraph";

export function emitBulletListItem(block: MDDMBlock, tokens: LayoutTokens): Paragraph[] {
  const runs = mddmTextRunsToDocxRuns(extractTextRuns(block), tokens);
  const options = { bullet: { level: 0 }, children: runs } as const;
  const paragraph = new Paragraph(options);
  (paragraph as unknown as { options: typeof options }).options = options;
  return [paragraph];
}

