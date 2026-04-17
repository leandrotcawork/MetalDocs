// Compile-time contract test. `tsc --noEmit` fails if types break.
import type {
  BreakCandidate, ComputedBreak, ServerBreak, ReconciledBreak,
  ReconcileLogs, ReconcileResult, PaginateRequest, PaginateResponse,
} from '../index';

// Exact-field assertions (add new field → must update contract or fail).
declare const bc: BreakCandidate;
const _bc: { afterBid: string; modelPath: readonly number[]; keepWithNext: boolean } = bc;
void _bc;
// Reverse: object literal excess property check — fails if index.ts adds undeclared field
const _bc2: BreakCandidate = { afterBid: '', modelPath: [], keepWithNext: false };
void _bc2;

declare const cb: ComputedBreak;
const _cb: { afterBid: string; pageNumber: number; yPx: number } = cb;
void _cb;
// Reverse: object literal excess property check — fails if index.ts adds undeclared field
const _cb2: ComputedBreak = { afterBid: '', pageNumber: 1, yPx: 0 };
void _cb2;

declare const sb: ServerBreak;
const _sb: { bid: string; pageNumber: number } = sb;
void _sb;
// Reverse: object literal excess property check — fails if index.ts adds undeclared field
const _sb2: ServerBreak = { bid: '', pageNumber: 1 };
void _sb2;

declare const rb: ReconciledBreak;
const _rb: { afterBid: string; pageNumber: number; source: 'editor' | 'editor-minor-drift' | 'server' } = rb;
void _rb;
// Reverse: object literal excess property check — fails if index.ts adds undeclared field
const _rb2: ReconciledBreak = { afterBid: '', pageNumber: 1, source: 'server' as const };
void _rb2;

declare const rl: ReconcileLogs;
const _rl: { exactMatches: number; minorDrift: number; majorDrift: number; orphanedEditor: number; serverOnly: number } = rl;
void _rl;
// Reverse: object literal excess property check — fails if index.ts adds undeclared field
const _rl2: ReconcileLogs = { exactMatches: 0, minorDrift: 0, majorDrift: 0, orphanedEditor: 0, serverOnly: 0 };
void _rl2;

declare const rr: ReconcileResult;
const _rr: { resolved: readonly ReconciledBreak[]; logs: ReconcileLogs } = rr;
void _rr;
// Reverse: object literal excess property check — fails if index.ts adds undeclared field
const _rr2: ReconcileResult = { resolved: [], logs: { exactMatches: 0, minorDrift: 0, majorDrift: 0, orphanedEditor: 0, serverOnly: 0 } };
void _rr2;

declare const pq: PaginateRequest;
const _pq: { html: string; editorBids?: readonly string[] } = pq;
void _pq;
// Reverse: object literal excess property check — fails if index.ts adds undeclared field
const _pq2: PaginateRequest = { html: '' };
void _pq2;

declare const pr: PaginateResponse;
const _pr: { breaks: readonly ServerBreak[] } = pr;
void _pr;
// Reverse: object literal excess property check — fails if index.ts adds undeclared field
const _pr2: PaginateResponse = { breaks: [] };
void _pr2;
