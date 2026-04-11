import { describe, expect, it } from "vitest";
import { TextRun } from "docx";
import { mddmTextRunsToDocxRuns } from "../inline-content";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMTextRun } from "../../../adapter";

describe("mddmTextRunsToDocxRuns", () => {
  it("emits a single TextRun for plain text", () => {
    const input: MDDMTextRun[] = [{ type: "text", text: "Hello world" }];
    const runs = mddmTextRunsToDocxRuns(input, defaultLayoutTokens);
    expect(runs).toHaveLength(1);
    expect(runs[0]).toBeInstanceOf(TextRun);
  });

  it("emits bold runs when marks include {type:'bold'}", () => {
    const input: MDDMTextRun[] = [{ type: "text", text: "Bold", marks: [{ type: "bold" }] }];
    const runs = mddmTextRunsToDocxRuns(input, defaultLayoutTokens);
    expect(runs).toHaveLength(1);
    expect((runs[0] as unknown as { options: any }).options).toMatchObject({ bold: true });
  });

  it("handles italic, underline, strike, and code marks", () => {
    const input: MDDMTextRun[] = [
      { type: "text", text: "x", marks: [{ type: "italic" }] },
      { type: "text", text: "y", marks: [{ type: "underline" }] },
      { type: "text", text: "z", marks: [{ type: "strike" }] },
      { type: "text", text: "c", marks: [{ type: "code" }] },
    ];
    const runs = mddmTextRunsToDocxRuns(input, defaultLayoutTokens);
    expect((runs[0] as unknown as { options: any }).options).toMatchObject({ italics: true });
    expect((runs[1] as unknown as { options: any }).options.underline).toBeDefined();
    expect((runs[2] as unknown as { options: any }).options).toMatchObject({ strike: true });
    expect((runs[3] as unknown as { options: any }).options.font).toBe("Consolas");
  });

  it("emits multiple runs for mixed marks", () => {
    const input: MDDMTextRun[] = [
      { type: "text", text: "Normal " },
      { type: "text", text: "bold", marks: [{ type: "bold" }] },
      { type: "text", text: " and " },
      { type: "text", text: "italic", marks: [{ type: "italic" }] },
    ];
    const runs = mddmTextRunsToDocxRuns(input, defaultLayoutTokens);
    expect(runs).toHaveLength(4);
  });

  it("honors exportFont and baseSizePt from tokens", () => {
    const input: MDDMTextRun[] = [{ type: "text", text: "Hi" }];
    const runs = mddmTextRunsToDocxRuns(input, defaultLayoutTokens);
    const options = (runs[0] as unknown as { options: any }).options;
    expect(options.font).toBe(defaultLayoutTokens.typography.exportFont);
    expect(options.size).toBe(defaultLayoutTokens.typography.baseSizePt * 2);
  });

  it("returns empty array for empty or undefined input", () => {
    expect(mddmTextRunsToDocxRuns([], defaultLayoutTokens)).toEqual([]);
    expect(mddmTextRunsToDocxRuns(undefined, defaultLayoutTokens)).toEqual([]);
  });

  it("ignores unknown marks without throwing", () => {
    const input: MDDMTextRun[] = [{ type: "text", text: "x", marks: [{ type: "unknown" }] }];
    expect(() => mddmTextRunsToDocxRuns(input, defaultLayoutTokens)).not.toThrow();
  });
});
