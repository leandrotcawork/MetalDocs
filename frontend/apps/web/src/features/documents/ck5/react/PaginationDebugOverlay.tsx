import type { ReconcileLogs } from "@metaldocs/mddm-pagination-types";

export interface PaginationDebugOverlayProps {
  logs: ReconcileLogs;
  debugFlag: boolean;
}

export function PaginationDebugOverlay({ logs, debugFlag }: PaginationDebugOverlayProps) {
  if (!debugFlag) {
    return null;
  }

  return (
    <div data-testid="pagination-debug-overlay">
      <span>exactMatches:{logs.exactMatches}</span>
      <span>minorDrift:{logs.minorDrift}</span>
      <span>majorDrift:{logs.majorDrift}</span>
      <span>orphanedEditor:{logs.orphanedEditor}</span>
      <span>serverOnly:{logs.serverOnly}</span>
    </div>
  );
}
