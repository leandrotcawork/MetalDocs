import type {
  ComputedBreak,
  ReconcileLogs,
  ReconcileResult,
  ReconciledBreak,
  ServerBreak,
} from "@metaldocs/mddm-pagination-types";

export type ParityReport = Readonly<{
  docId: string;
  editorBreaks: readonly ComputedBreak[];
  serverBreaks: readonly ServerBreak[];
  reconciled: readonly ReconciledBreak[];
  logs: ReconcileLogs;
  driftStats: Readonly<{
    totalBreaks: number;
    driftRatio: number;
  }>;
}>;

export function buildParityReport(
  docId: string,
  editorBreaks: readonly ComputedBreak[],
  serverBreaks: readonly ServerBreak[],
  reconcile: ReconcileResult,
): ParityReport {
  const totalBreaks = reconcile.resolved.length;
  const driftRatio = totalBreaks === 0
    ? 0
    : (reconcile.logs.minorDrift + reconcile.logs.majorDrift) / totalBreaks;

  return {
    docId,
    editorBreaks,
    serverBreaks,
    reconciled: reconcile.resolved,
    logs: reconcile.logs,
    driftStats: {
      totalBreaks,
      driftRatio,
    },
  };
}
