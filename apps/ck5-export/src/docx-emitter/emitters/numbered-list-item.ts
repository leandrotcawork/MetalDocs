import { Paragraph } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../shared/adapter";
import { mddmTextRunsToDocxRuns } from "../inline-content";
import { extractTextRuns } from "./paragraph";

// Stable string keeps OOXML output deterministic for golden tests.
export const MDDM_NUMBERING_REF = "mddm-decimal";

export function emitNumberedListItem(block: MDDMBlock, tokens: LayoutTokens): Paragraph[] {
  const runs = mddmTextRunsToDocxRuns(extractTextRuns(block), tokens);
  const options = {
    numbering: { reference: MDDM_NUMBERING_REF, level: 0 },
    children: runs,
  } as const;
  const paragraph = new Paragraph(options);
  (paragraph as unknown as { options: typeof options }).options = options;
  return [paragraph];
}

