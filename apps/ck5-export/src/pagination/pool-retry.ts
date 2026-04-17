export class WorkerCrash extends Error {
  constructor(m: string) { super(m); this.name = 'WorkerCrash'; }
}

export class PaginatorTimeoutError extends Error {
  constructor(public readonly ms: number) {
    super(`pagination timed out after ${ms}ms`);
    this.name = 'PaginatorTimeoutError';
  }
}

export type DegradedReason = 'worker-crash' | 'pool-exhausted' | 'runtime-error';

export class PaginationDegraded extends Error {
  public readonly reason: DegradedReason;
  constructor(reason: DegradedReason, cause?: unknown) {
    super(`pagination degraded: ${reason}`);
    this.name = 'PaginationDegraded';
    this.reason = reason;
    if (cause) (this as any).cause = cause;
  }
}

export const ACQUIRE_TIMEOUT_MS = 5_000;

type PoolLike = {
  acquire(): Promise<unknown>;
  release(w: unknown): void;
  replace(w: unknown): Promise<void>;
};

async function acquireWithTimeout(pool: PoolLike, timeoutMs: number): Promise<unknown> {
  let t: ReturnType<typeof setTimeout> | null = null;
  const timeout = new Promise<never>((_, rej) => {
    t = setTimeout(() => rej(new PaginationDegraded('pool-exhausted')), timeoutMs);
  });
  try {
    return await Promise.race([pool.acquire(), timeout]);
  } finally {
    if (t) clearTimeout(t);
  }
}

export async function withWorker<T>(
  pool: PoolLike,
  fn: (w: unknown) => Promise<T>,
  opts: { acquireTimeoutMs?: number } = {},
): Promise<T> {
  const timeoutMs = opts.acquireTimeoutMs ?? ACQUIRE_TIMEOUT_MS;
  let lastCrash: unknown = null;

  for (let attempt = 0; attempt < 2; attempt++) {
    const worker = await acquireWithTimeout(pool, timeoutMs);
    try {
      const result = await fn(worker);
      pool.release(worker);
      return result;
    } catch (e) {
      if (e instanceof PaginatorTimeoutError) {
        pool.release(worker);
        throw e;
      }
      if (e instanceof WorkerCrash) {
        lastCrash = e;
        await pool.replace(worker);
        continue; // retry once
      }
      // Unknown runtime error — graceful degraded fallback, no retry
      pool.release(worker);
      throw new PaginationDegraded('runtime-error', e);
    }
  }
  throw new PaginationDegraded('worker-crash', lastCrash);
}
