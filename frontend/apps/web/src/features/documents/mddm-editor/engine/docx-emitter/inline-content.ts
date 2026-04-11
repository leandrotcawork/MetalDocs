import { TextRun } from "docx";
import type { LayoutTokens } from "../layout-ir";
import { ptToHalfPt } from "../helpers/units";
import type { MDDMTextRun, MDDMMark } from "../../adapter";

const MARK_TYPES = new Set(["bold", "italic", "underline", "strike", "code"]);

function hasMark(marks: readonly MDDMMark[] | undefined, type: string): boolean {
  if (!marks || marks.length === 0) return false;
  return marks.some((m) => m?.type === type);
}

export function mddmTextRunsToDocxRuns(
  runs: readonly MDDMTextRun[] | undefined,
  tokens: LayoutTokens,
): TextRun[] {
  if (!runs || runs.length === 0) {
    return [];
  }

  const font = tokens.typography.exportFont;
  const baseHalfPt = ptToHalfPt(tokens.typography.baseSizePt);

  return runs.map((node) => {
    const marks = node.marks;
    const filteredMarks = (marks ?? []).filter((m) => MARK_TYPES.has(m?.type));
    const isCode = hasMark(filteredMarks, "code");

    const options = {
      text: node.text,
      font: isCode ? "Consolas" : font,
      size: baseHalfPt,
      bold: hasMark(filteredMarks, "bold"),
      italics: hasMark(filteredMarks, "italic"),
      underline: hasMark(filteredMarks, "underline") ? {} : undefined,
      strike: hasMark(filteredMarks, "strike"),
    } as const;

    const run = new TextRun(options);
    // Expose the raw options so tests and downstream helpers can introspect
    // inline formatting without reaching into docx.js internal OOXML nodes.
    (run as unknown as { options: typeof options }).options = options;
    return run;
  });
}
