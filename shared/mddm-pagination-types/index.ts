// Transport types shared across editor plugin and export service.
// Any change here MUST be accepted by both consumers.

export type BreakCandidate = Readonly<{
  afterBid: string;         // bid of block before the break
  modelPath: readonly number[]; // position path in CK5 model
  keepWithNext: boolean;    // true for headings → defer until next block fits
}>;

export type ComputedBreak = Readonly<{
  afterBid: string;
  pageNumber: number;       // 1-indexed
  yPx: number;              // cursor Y at break (debug)
}>;

export type ServerBreak = Readonly<{
  bid: string;              // matches afterBid from editor side
  pageNumber: number;
}>;

export type ReconciledBreak = Readonly<{
  afterBid: string;
  pageNumber: number;
  source: 'editor' | 'editor-minor-drift' | 'server';
}>;

export type ReconcileLogs = Readonly<{
  exactMatches: number;
  minorDrift: number;       // |delta|==1, editor honored
  majorDrift: number;       // |delta|>1, server wins
  orphanedEditor: number;   // editor bid not in server output
  serverOnly: number;       // server bid not in editor output
}>;

export type ReconcileResult = Readonly<{
  resolved: readonly ReconciledBreak[];
  logs: ReconcileLogs;
}>;

export type PaginateRequest = Readonly<{
  html: string;
  /** Optional: bids of breaks the editor reported. Enables editor-server-desync (422). */
  editorBids?: readonly string[];
}>;

export type PaginateResponse = Readonly<{
  breaks: readonly ServerBreak[];
}>;
