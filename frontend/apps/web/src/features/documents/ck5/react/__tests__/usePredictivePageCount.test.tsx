/**
 * @vitest-environment jsdom
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { usePredictivePageCount } from '../usePredictivePageCount';

const MM = (mm: number) => (mm / 25.4) * 96;
const STRIDE = MM(297) + 32;

class FakeResizeObserver {
  public static instances: FakeResizeObserver[] = [];
  public callback: ResizeObserverCallback;
  public constructor(cb: ResizeObserverCallback) {
    this.callback = cb;
    FakeResizeObserver.instances.push(this);
  }
  public observe = vi.fn();
  public unobserve = vi.fn();
  public disconnect = vi.fn();
  public fire(height: number): void {
    this.callback(
      [{ contentRect: { height } } as unknown as ResizeObserverEntry],
      this as unknown as ResizeObserver,
    );
  }
}

beforeEach(() => {
  FakeResizeObserver.instances = [];
  (globalThis as any).ResizeObserver = FakeResizeObserver;
});

describe('usePredictivePageCount', () => {
  it('defaults to 1 when editable null', () => {
    const { result } = renderHook(() => usePredictivePageCount(null));
    expect(result.current).toBe(1);
  });

  it('updates pages on resize — scrollHeight/stride ceil', () => {
    const el = document.createElement('div');
    Object.defineProperty(el, 'scrollHeight', { get: () => STRIDE * 2.1, configurable: true });
    el.style.setProperty = vi.fn();

    const { result } = renderHook(() => usePredictivePageCount(el));
    act(() => {
      FakeResizeObserver.instances[0].fire(STRIDE * 2.1);
    });
    expect(result.current).toBe(3);
    expect(el.style.setProperty).toHaveBeenCalledWith('--mddm-pages', '3');
  });

  it('minimum 1 page even for empty content — and writes CSS var', () => {
    const el = document.createElement('div');
    Object.defineProperty(el, 'scrollHeight', { get: () => 0, configurable: true });
    el.style.setProperty = vi.fn();

    const { result } = renderHook(() => usePredictivePageCount(el));
    act(() => {
      FakeResizeObserver.instances[0].fire(0);
    });
    expect(result.current).toBe(1);
    expect(el.style.setProperty).toHaveBeenCalledWith('--mddm-pages', '1');
  });
});
