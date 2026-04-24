import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { usePlaceholderValue } from '../usePlaceholderValue';
import * as api from '../../api/documentsV2';

vi.mock('../../api/documentsV2');

describe('usePlaceholderValue', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.mocked(api.putPlaceholderValue).mockResolvedValue(undefined);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it('initial value set correctly', () => {
    const { result } = renderHook(() => usePlaceholderValue('doc-1', 'p1', 'hello'));
    expect(result.current.value).toBe('hello');
  });

  it('setValue triggers save after 400ms debounce', async () => {
    const { result } = renderHook(() => usePlaceholderValue('doc-1', 'p1', ''));

    act(() => {
      result.current.setValue('new value');
    });

    expect(vi.mocked(api.putPlaceholderValue)).not.toHaveBeenCalled();

    await act(async () => {
      vi.advanceTimersByTime(400);
    });

    expect(vi.mocked(api.putPlaceholderValue)).toHaveBeenCalledWith('doc-1', 'p1', 'new value');
  });

  it('debounce resets on rapid changes', async () => {
    const { result } = renderHook(() => usePlaceholderValue('doc-1', 'p1', ''));

    act(() => { result.current.setValue('a'); });
    act(() => { result.current.setValue('ab'); });
    act(() => { result.current.setValue('abc'); });

    await act(async () => { vi.advanceTimersByTime(400); });

    expect(vi.mocked(api.putPlaceholderValue)).toHaveBeenCalledTimes(1);
    expect(vi.mocked(api.putPlaceholderValue)).toHaveBeenCalledWith('doc-1', 'p1', 'abc');
  });

  it('successful PUT clears error', async () => {
    vi.mocked(api.putPlaceholderValue).mockResolvedValue(undefined);
    const { result } = renderHook(() => usePlaceholderValue('doc-1', 'p1', ''));

    act(() => { result.current.setValue('ok'); });
    await act(async () => { vi.advanceTimersByTime(400); });

    expect(result.current.error).toBeNull();
  });

  it('422 response sets validation error message', async () => {
    const err422 = Object.assign(new Error('http_422'), {
      status: 422,
      body: JSON.stringify({ error: { message: 'regex mismatch', code: 'validation_failed' } }),
    });
    vi.mocked(api.putPlaceholderValue).mockRejectedValue(err422);

    const { result } = renderHook(() => usePlaceholderValue('doc-1', 'p1', ''));

    act(() => { result.current.setValue('bad'); });
    await act(async () => { vi.advanceTimersByTime(400); });

    expect(result.current.error).toBe('regex mismatch');
  });
});
