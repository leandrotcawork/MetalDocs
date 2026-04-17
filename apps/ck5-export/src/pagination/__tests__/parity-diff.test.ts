import { describe, expect, it } from "vitest";
import type {
  ComputedBreak,
  ReconcileLogs,
  ReconcileResult,
  ServerBreak,
} from "@metaldocs/mddm-pagination-types";
import { buildParityReport } from "../parity-diff";

describe("buildParityReport", () => {
  it("builds a report with zero drift ratio when there is only exact match", () => {
    const editorBreaks: readonly ComputedBreak[] = [
      { afterBid: "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", pageNumber: 1, yPx: 12 },
    ];
    const serverBreaks: readonly ServerBreak[] = [
      { bid: "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", pageNumber: 1 },
    ];
    const reconcile: ReconcileResult = {
      resolved: [
        {
          afterBid: "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa",
          pageNumber: 1,
          source: "editor",
        },
      ],
      logs: {
        exactMatches: 1,
        minorDrift: 0,
        majorDrift: 0,
        orphanedEditor: 0,
        serverOnly: 0,
      } satisfies ReconcileLogs,
    };

    const report = buildParityReport("doc-1", editorBreaks, serverBreaks, reconcile);

    expect(report).toMatchObject({
      docId: "doc-1",
      editorBreaks,
      serverBreaks,
      reconciled: reconcile.resolved,
      logs: reconcile.logs,
      driftStats: {
        totalBreaks: 1,
        driftRatio: 0,
      },
    });
  });
});
