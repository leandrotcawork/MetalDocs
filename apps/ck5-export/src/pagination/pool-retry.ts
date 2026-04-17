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
  const acquisition = pool.acquire();
  let settled = false;

  // If we time out, the pool waiter is still registered and will eventually
  // receive a worker. Release it back so it does not leak out of the pool.
  acquisition.then(
    (w) => { if (settled) pool.release(w); },
    () => { /* acquire error after timeout — nothing to release */ },
  );

  return new Promise<unknown>((resolve, reject) => {
    const t = setTimeout(() => {
      if (settled) return;
      settled = true;
      reject(new PaginationDegraded('pool-exhausted'));
    }, timeoutMs);
    acquisition.then(
      (w) => {
        if (settled) return;
        settled = true;
        clearTimeout(t);
        resolve(w);
      },
      (e) => {
        if (settled) return;
        settled = true;
        clearTimeout(t);
        reject(e);
      },
    );
  });
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
