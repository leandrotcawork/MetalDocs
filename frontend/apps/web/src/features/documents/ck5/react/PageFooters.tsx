import { createPortal } from 'react-dom';

const MM = (mm: number) => (mm / 25.4) * 96;
const STRIDE_PX = MM(297) + 32;
const FOOTER_Y_OFFSET_PX = MM(287);

const footerStyle = (pageIndex: number): React.CSSProperties => ({
  position: 'absolute',
  top: `${pageIndex * STRIDE_PX + FOOTER_Y_OFFSET_PX}px`,
  right: `${MM(20)}px`,
  color: '#6b7280',
  font: '11px/1 Carlito, "Liberation Sans", Arial, sans-serif',
  pointerEvents: 'none',
  zIndex: 1,
});

export interface PageFootersProps {
  pages: number;
  /** React-owned wrapper (e.g. `paperWrapperRef.current`). MUST NOT be the
   *  CKEditor DOM root — CK5's renderer reconciles its subtree and can remove
   *  non-editor children. Pass a plain div sized to the editor geometry. */
  portalTarget: HTMLElement | null;
}

export function PageFooters({ pages, portalTarget }: PageFootersProps): JSX.Element | null {
  if (!portalTarget) return null;
  const safeCount = Math.max(1, pages | 0);
  const nodes = Array.from({ length: safeCount }, (_, i) => (
    <div key={i} data-mddm-page-footer style={footerStyle(i)}>
      Page {i + 1}
    </div>
  ));
  return createPortal(<>{nodes}</>, portalTarget);
}
