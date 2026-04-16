import { Paragraph } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock, MDDMTextRun } from "../../shared/adapter";
import { mddmTextRunsToDocxRuns } from "../inline-content";

export function extractTextRuns(block: MDDMBlock): MDDMTextRun[] {
  const children = block.children ?? [];
  return (children as unknown[]).filter(
    (c): c is MDDMTextRun =>
      c !== null && typeof c === "object" && typeof (c as MDDMTextRun).text === "string",
  );
}

export function emitParagraph(block: MDDMBlock, tokens: LayoutTokens): Paragraph[] {
  const runs = mddmTextRunsToDocxRuns(extractTextRuns(block), tokens);
  const options = { children: runs } as const;
  const paragraph = new Paragraph(options);
  (paragraph as unknown as { options: typeof options }).options = options;
  return [paragraph];
}

