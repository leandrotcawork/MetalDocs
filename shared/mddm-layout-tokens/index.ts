export type LayoutTokens = Readonly<{
  page: Readonly<{
    widthMm: number;
    heightMm: number;
    marginTopMm: number;
    marginBottomMm: number;
    marginLeftMm: number;
    marginRightMm: number;
  }>;
  typography: Readonly<{
    exportFont: string;
    baseFontSizePt: number;
  }>;
  paginationSLO: Readonly<{
    maxBreakDeltaPer50Pages: number;
  }>;
  blockIdentityAttr: string;
  pageBreakAttr: string;
  fontFallbackChain: readonly string[];
}>;

export const defaultLayoutTokens: LayoutTokens = {
  page: {
    widthMm: 210,
    heightMm: 297,
    marginTopMm: 25,
    marginBottomMm: 25,
    marginLeftMm: 25,
    marginRightMm: 25,
  },
  typography: {
    exportFont: 'Carlito',
    baseFontSizePt: 11,
  },
  paginationSLO: { maxBreakDeltaPer50Pages: 1 },
  blockIdentityAttr: 'data-mddm-bid',
  pageBreakAttr: 'data-pagination-page',
  fontFallbackChain: Object.freeze(['Carlito', 'Liberation Sans', 'Arial', 'sans-serif']) as readonly string[],
} as const;
