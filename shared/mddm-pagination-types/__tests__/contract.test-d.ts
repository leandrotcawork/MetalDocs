// Compile-time contract test. `tsc --noEmit` fails if types break.
import type {
  BreakCandidate, ComputedBreak, ServerBreak, ReconciledBreak,
  ReconcileLogs, ReconcileResult, PaginateRequest, PaginateResponse,
} from '../index';

// Exact-field assertions (add new field → must update contract or fail).
declare const bc: BreakCandidate;
const _bc: { afterBid: string; modelPath: readonly number[]; keepWithNext: boolean } = bc;
void _bc;

declare const cb: ComputedBreak;
const _cb: { afterBid: string; pageNumber: number; yPx: number } = cb;
void _cb;

declare const sb: ServerBreak;
const _sb: { bid: string; pageNumber: number } = sb;
void _sb;

declare const rb: ReconciledBreak;
const _rb: { afterBid: string; pageNumber: number; source: 'editor' | 'editor-minor-drift' | 'server' } = rb;
void _rb;

declare const rl: ReconcileLogs;
const _rl: { exactMatches: number; minorDrift: number; majorDrift: number; orphanedEditor: number; serverOnly: number } = rl;
void _rl;

declare const rr: ReconcileResult;
const _rr: { resolved: readonly ReconciledBreak[]; logs: ReconcileLogs } = rr;
void _rr;

declare const pq: PaginateRequest;
const _pq: { html: string; editorBids?: readonly string[] } = pq;
void _pq;

declare const pr: PaginateResponse;
const _pr: { breaks: readonly ServerBreak[] } = pr;
void _pr;
