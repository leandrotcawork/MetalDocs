import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useDocumentAutosave, type AutosaveArgs } from './useDocumentAutosave';
import * as api from '../api/documentsV2';
import * as idb from './useIndexedDBRestore';

vi.mock('../api/documentsV2');
vi.mock('./useIndexedDBRestore', () => ({
  putPending: vi.fn().mockResolvedValue(undefined),
  deletePending: vi.fn().mockResolvedValue(undefined),
  getAllPending: vi.fn().mockResolvedValue([]),
}));

// crypto.subtle stub for jsdom
Object.defineProperty(globalThis, 'crypto', {
  value: { subtle: { digest: vi.fn().mockResolvedValue(new Uint8Array(32).buffer) } },
});

const baseArgs = (): AutosaveArgs => ({
  documentID: 'doc-1',
  sessionID: 'sess-1',
  baseRevisionID: 'rev-0',
  onAdvanceBase: vi.fn(),
  onSessionLost: vi.fn(),
});

describe('useDocumentAutosave', () => {
  beforeEach(() => {
    vi.mocked(api.presignAutosave).mockResolvedValue({ UploadURL: 'http://s3/upload', PendingUploadID: 'pend-1', ExpiresAt: '' });
    vi.mocked(api.commitAutosave).mockResolvedValue({ revision_id: 'rev-1', revision_num: 2 });
    global.fetch = vi.fn().mockResolvedValue({ ok: true } as any);
  });
  afterEach(() => vi.clearAllMocks());

  it('queue -> flush -> saved; onAdvanceBase called', async () => {
    const args = baseArgs();
    const { result } = renderHook(() => useDocumentAutosave(args));
    await act(async () => {
      await result.current.queue(new ArrayBuffer(4), { field: 'val' });
      await result.current.flush();
    });
    expect(result.current.status).toBe('saved');
    expect(args.onAdvanceBase).toHaveBeenCalledWith('rev-1');
  });

  it('409 stale_base -> onSessionLost stale_base, status stale', async () => {
    const args = baseArgs();
    vi.mocked(api.presignAutosave).mockRejectedValueOnce(
      Object.assign(new Error(), { status: 409, body: JSON.stringify({ error: 'stale_base' }) })
    );
    const { result } = renderHook(() => useDocumentAutosave(args));
    await act(async () => {
      await result.current.queue(new ArrayBuffer(4), null);
      await result.current.flush();
    });
    expect(args.onSessionLost).toHaveBeenCalledWith('stale_base');
    expect(result.current.status).toBe('stale');
  });

  it('409 session_inactive -> onSessionLost session_inactive, status session_lost', async () => {
    const args = baseArgs();
    vi.mocked(api.presignAutosave).mockRejectedValueOnce(
      Object.assign(new Error(), { status: 409, body: JSON.stringify({ error: 'session_inactive' }) })
    );
    const { result } = renderHook(() => useDocumentAutosave(args));
    await act(async () => {
      await result.current.queue(new ArrayBuffer(4), null);
      await result.current.flush();
    });
    expect(args.onSessionLost).toHaveBeenCalledWith('session_inactive');
    expect(result.current.status).toBe('session_lost');
  });

  it('409 session_not_holder → onSessionLost session_inactive, status session_lost', async () => {
    const args = baseArgs();
    vi.mocked(api.presignAutosave).mockRejectedValueOnce(
      Object.assign(new Error(), { status: 409, body: JSON.stringify({ error: 'session_not_holder' }) })
    );
    const { result } = renderHook(() => useDocumentAutosave(args));
    await act(async () => {
      await result.current.queue(new ArrayBuffer(4), null);
      await result.current.flush();
    });
    expect(args.onSessionLost).toHaveBeenCalledWith('session_inactive');
    expect(result.current.status).toBe('session_lost');
  });

  it('410 upload_missing -> status error, IndexedDB deleted, pending cleared', async () => {
    const args = baseArgs();
    vi.mocked(api.commitAutosave).mockRejectedValueOnce(
      Object.assign(new Error(), { status: 410, body: JSON.stringify({ error: 'upload_missing' }) })
    );
    const { result } = renderHook(() => useDocumentAutosave(args));
    await act(async () => {
      await result.current.queue(new ArrayBuffer(4), null);
      await result.current.flush();
    });
    expect(result.current.status).toBe('error');
    expect(vi.mocked(idb.deletePending)).toHaveBeenCalledWith('doc-1', expect.any(String));
    expect(vi.mocked(idb.getAllPending)).toBeDefined(); // verifies import wired
  });

  it('410 expired_upload -> status error, pending cleared', async () => {
    const args = baseArgs();
    vi.mocked(api.commitAutosave).mockRejectedValueOnce(
      Object.assign(new Error(), { status: 410, body: JSON.stringify({ error: 'expired_upload' }) })
    );
    const { result } = renderHook(() => useDocumentAutosave(args));
    await act(async () => {
      await result.current.queue(new ArrayBuffer(4), null);
      await result.current.flush();
    });
    expect(result.current.status).toBe('error');
    expect(vi.mocked(idb.deletePending)).toHaveBeenCalled();
  });

  it('422 content_hash_mismatch -> status error, pending cleared', async () => {
    const args = baseArgs();
    vi.mocked(api.commitAutosave).mockRejectedValueOnce(
      Object.assign(new Error(), { status: 422, body: JSON.stringify({ error: 'content_hash_mismatch' }) })
    );
    const { result } = renderHook(() => useDocumentAutosave(args));
    await act(async () => {
      await result.current.queue(new ArrayBuffer(4), null);
      await result.current.flush();
    });
    expect(result.current.status).toBe('error');
    expect(vi.mocked(idb.deletePending)).toHaveBeenCalled();
  });

  it('replays IndexedDB leftover on mount if session matches', async () => {
    const args = baseArgs();
    const leftover = {
      document_id: 'doc-1', session_id: 'sess-1', base_revision_id: 'rev-0',
      content_hash: 'abc', buffer: new ArrayBuffer(4), created_at: Date.now(),
    };
    vi.mocked(idb.getAllPending).mockResolvedValueOnce([leftover as any]);
    renderHook(() => useDocumentAutosave(args));
    // Recovery runs asynchronously on mount -- verify commit was called
    await vi.waitFor(() => expect(api.commitAutosave).toHaveBeenCalled());
    expect(idb.deletePending).toHaveBeenCalledWith('doc-1', 'abc');
  });
});
