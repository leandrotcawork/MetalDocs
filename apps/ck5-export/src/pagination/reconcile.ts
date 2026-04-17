type EditorBreak = Readonly<{ afterBid: string; pageNumber: number }>;
type ServerBreak = Readonly<{ bid: string; pageNumber: number }>;
export type ReconciledBreak = Readonly<{ afterBid: string; pageNumber: number; source: 'editor' | 'editor-minor-drift' | 'server' }>;
export type ReconcileResult = Readonly<{
  resolved: readonly ReconciledBreak[];
  logs: Readonly<{
    exactMatches: number;
    minorDrift: number;
    majorDrift: number;
    orphanedEditor: number;
    serverOnly: number;
  }>;
}>;

export function reconcile(
  editor: readonly EditorBreak[],
  server: readonly ServerBreak[],
): ReconcileResult {
  const serverByBid = new Map(server.map(s => [s.bid, s]));
  const editorByBid = new Map(editor.map(e => [e.afterBid, e]));

  const resolved: ReconciledBreak[] = [];
  let exactMatches = 0, minorDrift = 0, majorDrift = 0, orphanedEditor = 0, serverOnly = 0;

  for (const e of editor) {
    const s = serverByBid.get(e.afterBid);
    if (!s) { orphanedEditor++; continue; }
    const delta = Math.abs(e.pageNumber - s.pageNumber);
    if (delta === 0) { resolved.push({ afterBid: e.afterBid, pageNumber: e.pageNumber, source: 'editor' }); exactMatches++; }
    else if (delta <= 1) { resolved.push({ afterBid: e.afterBid, pageNumber: e.pageNumber, source: 'editor-minor-drift' }); minorDrift++; }
    else { resolved.push({ afterBid: e.afterBid, pageNumber: s.pageNumber, source: 'server' }); majorDrift++; }
  }
  for (const s of server) {
    if (!editorByBid.has(s.bid)) {
      resolved.push({ afterBid: s.bid, pageNumber: s.pageNumber, source: 'server' });
      serverOnly++;
    }
  }

  resolved.sort((a, b) => a.pageNumber - b.pageNumber);
  return {
    resolved,
    logs: { exactMatches, minorDrift, majorDrift, orphanedEditor, serverOnly },
  };
}
