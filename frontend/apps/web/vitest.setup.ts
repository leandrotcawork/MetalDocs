import '@testing-library/jest-dom/vitest';

// ResizeObserver — CK5 UI components observe editable resize.
if (typeof ResizeObserver === 'undefined') {
  class RO { observe() {} unobserve() {} disconnect() {} }
  (globalThis as unknown as { ResizeObserver: typeof RO }).ResizeObserver = RO;
}

// IntersectionObserver — used by sticky toolbar logic.
if (typeof IntersectionObserver === 'undefined') {
  class IO {
    observe() {}
    unobserve() {}
    disconnect() {}
    takeRecords() { return []; }
    root = null;
    rootMargin = '';
    thresholds = [] as readonly number[];
  }
  (globalThis as unknown as { IntersectionObserver: typeof IO }).IntersectionObserver = IO;
}

// matchMedia — CK5 responsive layout queries.
if (typeof window !== 'undefined' && !window.matchMedia) {
  window.matchMedia = (query) => ({
    matches: false, media: query, onchange: null,
    addListener() {}, removeListener() {}, addEventListener() {}, removeEventListener() {},
    dispatchEvent() { return false; },
  });
}

// document.createRange exists in jsdom but missing getBoundingClientRect on ranges.
if (typeof document !== 'undefined') {
  const originalCreateRange = document.createRange.bind(document);
  document.createRange = () => {
    const range = originalCreateRange();
    if (typeof range.getBoundingClientRect !== 'function') {
      range.getBoundingClientRect = () => ({
        x: 0, y: 0, top: 0, left: 0, bottom: 0, right: 0, width: 0, height: 0,
        toJSON() { return {}; },
      });
    }
    if (typeof (range as unknown as { getClientRects?: () => unknown }).getClientRects !== 'function') {
      (range as unknown as { getClientRects: () => unknown[] }).getClientRects = () => [];
    }
    return range;
  };
}

// Clipboard API is used by CK5 clipboard pipeline.
if (typeof navigator !== 'undefined' && !navigator.clipboard) {
  Object.defineProperty(navigator, 'clipboard', {
    value: { writeText: async () => {}, readText: async () => '' },
    configurable: true,
  });
}
