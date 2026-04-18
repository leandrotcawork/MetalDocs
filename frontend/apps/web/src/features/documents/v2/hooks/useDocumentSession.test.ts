import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useDocumentSession } from './useDocumentSession';
import * as api from '../api/documentsV2';

vi.mock('../api/documentsV2');

describe('useDocumentSession', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.mocked(api.acquireSession).mockResolvedValue({
      mode: 'writer',
      session_id: 'sess-1',
      expires_at: '2099-01-01T00:00:00Z',
      last_ack_revision_id: 'rev-1',
    });
    vi.mocked(api.heartbeatSession).mockResolvedValue({} as any);
  });
  afterEach(() => { vi.useRealTimers(); vi.clearAllMocks(); });

  it('acquires writer session and starts heartbeat', async () => {
    const { result } = renderHook(() => useDocumentSession('doc-1'));

    await waitFor(() => expect(result.current.state.phase).toBe('writer'));
    expect(api.acquireSession).toHaveBeenCalledWith('doc-1');

    // Advance 30s -> heartbeat fires
    await act(async () => { vi.advanceTimersByTime(30_000); });
    await waitFor(() => expect(api.heartbeatSession).toHaveBeenCalledWith('doc-1', 'sess-1'));
  });

  it('transitions to lost:force_released when heartbeat returns 409', async () => {
    const { result } = renderHook(() => useDocumentSession('doc-1'));
    await waitFor(() => expect(result.current.state.phase).toBe('writer'));

    vi.mocked(api.heartbeatSession).mockRejectedValueOnce(
      Object.assign(new Error('http_409'), { status: 409 })
    );
    await act(async () => { vi.advanceTimersByTime(30_000); });
    await waitFor(() => expect(result.current.state).toEqual({ phase: 'lost', reason: 'force_released' }));
  });
});
