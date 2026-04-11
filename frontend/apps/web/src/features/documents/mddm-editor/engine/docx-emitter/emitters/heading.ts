import { Paragraph, HeadingLevel } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { mddmTextRunsToDocxRuns } from "../inline-content";
import { extractTextRuns } from "./paragraph";

type HeadingLevelValue = (typeof HeadingLevel)[keyof typeof HeadingLevel];

function levelToHeading(level: unknown): HeadingLevelValue {
  const n = typeof level === "number" ? level : Number(level);
  switch (n) {
    case 1:
      return HeadingLevel.HEADING_1;
    case 2:
      return HeadingLevel.HEADING_2;
    case 3:
      return HeadingLevel.HEADING_3;
    default:
      return HeadingLevel.HEADING_1;
  }
}

export function emitHeading(block: MDDMBlock, tokens: LayoutTokens): Paragraph[] {
  const runs = mddmTextRunsToDocxRuns(extractTextRuns(block), tokens);
  const options = {
    heading: levelToHeading((block.props as { level?: unknown }).level),
    children: runs,
  } as const;
  const paragraph = new Paragraph(options);
  (paragraph as unknown as { options: typeof options }).options = options;
  return [paragraph];
}
