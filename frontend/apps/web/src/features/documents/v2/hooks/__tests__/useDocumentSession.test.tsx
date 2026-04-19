import React, { type PropsWithChildren } from 'react';
import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useDocumentSession } from '../useDocumentSession';
import * as api from '../../api/documentsV2';

vi.mock('../../api/documentsV2');

const WRITER_ONE = {
  mode: 'writer' as const,
  session_id: 'sess-1',
  expires_at: '2099-01-01T00:00:00Z',
  last_ack_revision_id: 'rev-1',
};

const WRITER_TWO = {
  mode: 'writer' as const,
  session_id: 'sess-2',
  expires_at: '2099-01-01T00:00:00Z',
  last_ack_revision_id: 'rev-2',
};

describe('useDocumentSession (phase 3)', () => {
  let hidden = false;

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-04-19T00:00:00.000Z'));
    hidden = false;
    Object.defineProperty(document, 'hidden', {
      configurable: true,
      get: () => hidden,
    });

    vi.mocked(api.acquireSession).mockResolvedValue(WRITER_ONE);
    vi.mocked(api.heartbeatSession).mockResolvedValue({} as never);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('guards StrictMode double-mount so acquire runs once', async () => {
    let resolveAcquire!: (value: typeof WRITER_ONE) => void;
    const pendingAcquire = new Promise<typeof WRITER_ONE>((resolve) => {
      resolveAcquire = resolve;
    });
    vi.mocked(api.acquireSession).mockReturnValueOnce(pendingAcquire);

    const wrapper = ({ children }: PropsWithChildren) => <React.StrictMode>{children}</React.StrictMode>;
    const { unmount } = renderHook(() => useDocumentSession('doc-1'), { wrapper });

    expect(api.acquireSession).toHaveBeenCalledTimes(1);

    resolveAcquire(WRITER_ONE);
    await act(async () => {
      await pendingAcquire;
    });

    unmount();
  });

  it('heartbeat 409 performs silent retry and stays writer', async () => {
    vi.mocked(api.acquireSession).mockResolvedValueOnce(WRITER_ONE).mockResolvedValueOnce(WRITER_TWO);
    vi.mocked(api.heartbeatSession)
      .mockRejectedValueOnce(Object.assign(new Error('conflict'), { status: 409 }))
      .mockResolvedValue({} as never);

    const { result } = renderHook(() => useDocumentSession('doc-1'));

    await waitFor(() => expect(result.current.state.phase).toBe('writer'));

    await act(async () => {
      vi.advanceTimersByTime(30_000);
    });

    await waitFor(() => expect(api.acquireSession).toHaveBeenCalledTimes(2));
    expect(result.current.state.phase).toBe('writer');

    await act(async () => {
      vi.advanceTimersByTime(30_000);
    });

    await waitFor(() => expect(api.heartbeatSession).toHaveBeenLastCalledWith('doc-1', 'sess-2'));
    expect(result.current.state.phase).toBe('writer');
  });

  it('heartbeat 409 twice in a row transitions to lost', async () => {
    vi.mocked(api.acquireSession).mockResolvedValueOnce(WRITER_ONE).mockResolvedValueOnce(WRITER_TWO);
    vi.mocked(api.heartbeatSession).mockRejectedValue(Object.assign(new Error('conflict'), { status: 409 }));

    const { result } = renderHook(() => useDocumentSession('doc-1'));
    await waitFor(() => expect(result.current.state.phase).toBe('writer'));

    await act(async () => {
      vi.advanceTimersByTime(30_000);
    });

    await waitFor(() => expect(api.acquireSession).toHaveBeenCalledTimes(2));
    expect(result.current.state.phase).toBe('writer');

    await act(async () => {
      vi.advanceTimersByTime(30_000);
    });

    await waitFor(() => expect(result.current.state).toEqual({ phase: 'lost', reason: 'force_released' }));
  });

  it('re-acquires when tab is hidden for more than two minutes', async () => {
    vi.mocked(api.acquireSession).mockResolvedValueOnce(WRITER_ONE).mockResolvedValueOnce(WRITER_TWO);

    renderHook(() => useDocumentSession('doc-1'));

    await waitFor(() => expect(api.acquireSession).toHaveBeenCalledTimes(1));

    hidden = true;
    act(() => {
      document.dispatchEvent(new Event('visibilitychange'));
    });

    await act(async () => {
      vi.advanceTimersByTime(120_001);
    });

    hidden = false;
    act(() => {
      document.dispatchEvent(new Event('visibilitychange'));
    });

    await waitFor(() => expect(api.acquireSession).toHaveBeenCalledTimes(2));
  });
});
