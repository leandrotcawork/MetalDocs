import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";

type ResizableSplitPaneProps = {
  left: ReactNode;
  right: ReactNode;
  storageKey?: string;
  defaultRightWidth?: number;
  minLeftWidth?: number;
  minRightWidth?: number;
};

export function ResizableSplitPane({
  left,
  right,
  storageKey = "metaldocs:editor-split-width",
  defaultRightWidth = 420,
  minLeftWidth = 400,
  minRightWidth = 340,
}: ResizableSplitPaneProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [rightWidth, setRightWidth] = useState(() => {
    try {
      const stored = localStorage.getItem(storageKey);
      if (stored) {
        const parsed = Number(stored);
        if (!Number.isNaN(parsed) && parsed >= minRightWidth) return parsed;
      }
    } catch { /* ignore */ }
    return defaultRightWidth;
  });
  const isDragging = useRef(false);

  const persistWidth = useCallback((width: number) => {
    try {
      localStorage.setItem(storageKey, String(Math.round(width)));
    } catch { /* ignore */ }
  }, [storageKey]);

  const handlePointerDown = useCallback((e: React.PointerEvent) => {
    e.preventDefault();
    isDragging.current = true;
    (e.target as HTMLElement).setPointerCapture(e.pointerId);
  }, []);

  const handlePointerMove = useCallback((e: React.PointerEvent) => {
    if (!isDragging.current || !containerRef.current) return;
    const containerRect = containerRef.current.getBoundingClientRect();
    const newRight = containerRect.right - e.clientX;
    const maxRight = containerRect.width - minLeftWidth;
    const clamped = Math.max(minRightWidth, Math.min(maxRight, newRight));
    setRightWidth(clamped);
  }, [minLeftWidth, minRightWidth]);

  const handlePointerUp = useCallback((e: React.PointerEvent) => {
    if (!isDragging.current) return;
    isDragging.current = false;
    (e.target as HTMLElement).releasePointerCapture(e.pointerId);
    setRightWidth((w) => {
      persistWidth(w);
      return w;
    });
  }, [persistWidth]);

  const handleDoubleClick = useCallback(() => {
    setRightWidth(defaultRightWidth);
    persistWidth(defaultRightWidth);
  }, [defaultRightWidth, persistWidth]);

  useEffect(() => {
    const handleSelectStart = (e: Event) => {
      if (isDragging.current) e.preventDefault();
    };
    document.addEventListener("selectstart", handleSelectStart);
    return () => document.removeEventListener("selectstart", handleSelectStart);
  }, []);

  return (
    <div className="split-pane" ref={containerRef}>
      <div className="split-pane-left" style={{ flex: 1, minWidth: minLeftWidth }}>
        {left}
      </div>
      <div
        className="split-handle"
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
        onDoubleClick={handleDoubleClick}
        role="separator"
        aria-orientation="vertical"
        tabIndex={0}
      />
      <div className="split-pane-right" style={{ width: rightWidth, minWidth: minRightWidth, flexShrink: 0 }}>
        {right}
      </div>
    </div>
  );
}
