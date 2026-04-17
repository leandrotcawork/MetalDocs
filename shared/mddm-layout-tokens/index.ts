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
} as const;
