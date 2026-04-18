import { useEffect, useState } from 'react';

const MM_PER_INCH = 25.4;
const PX_PER_INCH = 96;
const PAGE_HEIGHT_PX = (297 / MM_PER_INCH) * PX_PER_INCH;
const PAGE_GAP_PX = 32;
const STRIDE_PX = PAGE_HEIGHT_PX + PAGE_GAP_PX;

export function usePredictivePageCount(editable: HTMLElement | null): number {
  const [pages, setPages] = useState(1);

  useEffect(() => {
    if (!editable) return;
    if (typeof ResizeObserver === 'undefined') {
      return;
    }
    const compute = () => {
      const height = editable.scrollHeight;
      const next = Math.max(1, Math.ceil(height / STRIDE_PX));
      editable.style.setProperty('--mddm-pages', String(next));
      setPages((prev) => (prev !== next ? next : prev));
    };
    const ro = new ResizeObserver(compute);
    ro.observe(editable);
    compute();
    return () => ro.disconnect();
  }, [editable]);

  return pages;
}
