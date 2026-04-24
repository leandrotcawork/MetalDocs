import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useComputedRefresh } from '../useComputedRefresh';
import * as api from '../../api/documentsV2';

vi.mock('../../api/documentsV2');

describe('useComputedRefresh', () => {
  beforeEach(() => {
    vi.mocked(api.getPlaceholderValues).mockResolvedValue([
      { placeholder_id: 'p1', value_text: 'computed-val', source: 'computed' },
    ]);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('triggerRefresh calls getPlaceholderValues', async () => {
    const onRefreshed = vi.fn();
    const { result } = renderHook(() => useComputedRefresh('doc-1', onRefreshed));

    await act(async () => {
      result.current.triggerRefresh();
    });

    expect(vi.mocked(api.getPlaceholderValues)).toHaveBeenCalledWith('doc-1');
  });

  it('triggerRefresh passes values to onRefreshed', async () => {
    const onRefreshed = vi.fn();
    const { result } = renderHook(() => useComputedRefresh('doc-1', onRefreshed));

    await act(async () => {
      result.current.triggerRefresh();
    });

    expect(onRefreshed).toHaveBeenCalledWith([
      { placeholder_id: 'p1', value_text: 'computed-val', source: 'computed' },
    ]);
  });

  it('refreshing goes true then false', async () => {
    let resolve!: () => void;
    vi.mocked(api.getPlaceholderValues).mockReturnValue(
      new Promise<typeof api.getPlaceholderValues extends (...args: any[]) => Promise<infer R> ? R : never>((r) => {
        resolve = () => r([]);
      }),
    );
    const { result } = renderHook(() => useComputedRefresh('doc-1', vi.fn()));

    act(() => { result.current.triggerRefresh(); });
    expect(result.current.refreshing).toBe(true);

    await act(async () => { resolve(); });
    expect(result.current.refreshing).toBe(false);
  });
});
