// Re-export wire types — single source of truth.
export type {
  BreakCandidate,
  ComputedBreak,
  ServerBreak,
  ReconciledBreak,
  ReconcileLogs,
  ReconcileResult,
  PaginateRequest,
  PaginateResponse,
} from '@metaldocs/mddm-pagination-types';

// CK5-engine-tied internal — NOT a wire type. Never crosses to Node.
import type { Position as ModelPosition } from 'ckeditor5';
export type InternalBreakAnchor = Readonly<{
  bid: string;
  position: ModelPosition;
}>;
