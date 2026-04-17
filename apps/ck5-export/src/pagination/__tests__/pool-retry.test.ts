import { describe, it, expect, vi } from 'vitest';
import { withWorker, WorkerCrash, PaginationDegraded, PaginatorTimeoutError } from '../pool-retry';

function fakePool(): any {
  const w1 = { id: 'w1' };
  const w2 = { id: 'w2' };
  const workers = [w1, w2];
  let next = 0;
  return {
    acquire: vi.fn(async () => workers[next++ % workers.length]),
    release: vi.fn(),
    replace: vi.fn(async () => {}),
  };
}

describe('withWorker', () => {
  it('returns fn result when first attempt succeeds', async () => {
    const pool = fakePool();
    const r = await withWorker(pool, async () => 'ok');
    expect(r).toBe('ok');
    expect(pool.acquire).toHaveBeenCalledTimes(1);
  });

  it('retries once on WorkerCrash and succeeds', async () => {
    const pool = fakePool();
    let n = 0;
    const r = await withWorker(pool, async () => {
      if (n++ === 0) throw new WorkerCrash('boom');
      return 'ok';
    });
    expect(r).toBe('ok');
    expect(pool.acquire).toHaveBeenCalledTimes(2);
    expect(pool.replace).toHaveBeenCalledTimes(1);
  });

  it('rejects PaginationDegraded on second crash', async () => {
    const pool = fakePool();
    await expect(withWorker(pool, async () => { throw new WorkerCrash('boom'); }))
      .rejects.toBeInstanceOf(PaginationDegraded);
  });

  it('propagates PaginatorTimeoutError without retry', async () => {
    const pool = fakePool();
    await expect(withWorker(pool, async () => { throw new PaginatorTimeoutError(5000); }))
      .rejects.toBeInstanceOf(PaginatorTimeoutError);
    expect(pool.acquire).toHaveBeenCalledTimes(1);
  });

  it('rejects PaginationDegraded when acquire exceeds acquireTimeoutMs', async () => {
    const pool = { acquire: vi.fn(() => new Promise(() => {})), release: vi.fn(), replace: vi.fn() } as any;
    const err = await withWorker(pool, async () => 'x', { acquireTimeoutMs: 50 }).catch(e => e);
    expect(err).toBeInstanceOf(PaginationDegraded);
    expect(err.reason).toBe('pool-exhausted');
  });

  it('rejects PaginationDegraded reason:runtime-error on unknown error (no retry)', async () => {
    const pool = fakePool();
    const err = await withWorker(pool, async () => { throw new Error('boom'); }).catch(e => e);
    expect(err).toBeInstanceOf(PaginationDegraded);
    expect(err.reason).toBe('runtime-error');
    expect(pool.acquire).toHaveBeenCalledTimes(1);
  });
});
