import { act } from 'react';
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { PageCounter } from '../PageCounter';

function makeFakeEditor() {
  type Cb = (b: { afterBid: string; pageNumber: number }[]) => void;
  let cb: Cb | null = null;
  const off = vi.fn(() => {
    cb = null;
  });
  const onBreaks = vi.fn((fn: Cb) => {
    cb = fn;
    return off;
  });
  const plugin = { _measurer: { onBreaks } };
  const editor: any = {
    plugins: { get: (n: string) => (n === 'MddmPagination' ? plugin : null) }
  };
  return { editor, fire: (b: Parameters<Cb>[0]) => cb?.(b), off };
}

describe('PageCounter', () => {
  it('renders "Page 1 of 1" with no breaks', () => {
    const { editor } = makeFakeEditor();
    render(<PageCounter editor={editor} />);
    expect(screen.getByText(/Page 1 of 1/)).toBeTruthy();
  });

  it('updates when measurer emits breaks', () => {
    const { editor, fire } = makeFakeEditor();
    render(<PageCounter editor={editor} />);
    act(() => {
      fire([
        { afterBid: 'a', pageNumber: 2 },
        { afterBid: 'b', pageNumber: 3 }
      ]);
    });
    expect(screen.getByText(/Page 3 of 3/)).toBeTruthy();
  });

  it('calls off() on unmount', () => {
    const { editor, off } = makeFakeEditor();
    const { unmount } = render(<PageCounter editor={editor} />);
    unmount();
    expect(off).toHaveBeenCalled();
  });
});
