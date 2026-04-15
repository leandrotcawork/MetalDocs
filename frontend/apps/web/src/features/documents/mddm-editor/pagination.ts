export type PaginationBlock = {
  id: string;
  topPx: number;
  heightPx: number;
};

export type ComputePageLayoutInput = {
  pageHeightPx: number;
  topMarginPx: number;
  bottomMarginPx: number;
  blocks: ReadonlyArray<PaginationBlock>;
};

export type PageLayout = {
  pageCount: number;
  breakOffsetsByBlockId: Record<string, number>;
};

type NormalizedBlock = {
  id: string;
  topPx: number;
  heightPx: number;
  bottomPx: number;
};

function toNonNegativeFinite(value: number): number {
  if (!Number.isFinite(value)) {
    return 0;
  }
  return Math.max(0, value);
}

function normalizeBlock(block: PaginationBlock): NormalizedBlock {
  const topPx = toNonNegativeFinite(block.topPx);
  const heightPx = toNonNegativeFinite(block.heightPx);
  return {
    id: block.id,
    topPx,
    heightPx,
    bottomPx: topPx + heightPx,
  };
}

function compareBlocks(a: NormalizedBlock, b: NormalizedBlock): number {
  if (a.topPx !== b.topPx) return a.topPx - b.topPx;
  if (a.bottomPx !== b.bottomPx) return a.bottomPx - b.bottomPx;
  return a.id.localeCompare(b.id);
}

function computeWritableHeightPx(input: ComputePageLayoutInput): number {
  const pageHeightPx = toNonNegativeFinite(input.pageHeightPx);
  const topMarginPx = toNonNegativeFinite(input.topMarginPx);
  const bottomMarginPx = toNonNegativeFinite(input.bottomMarginPx);
  return Math.max(1, pageHeightPx - topMarginPx - bottomMarginPx);
}

export function computePageLayout(input: ComputePageLayoutInput): PageLayout {
  const writableHeightPx = computeWritableHeightPx(input);
  const blocks = input.blocks.map(normalizeBlock).sort(compareBlocks);
  const breakOffsetsByBlockId: Record<string, number> = {};

  for (const block of blocks) {
    if (block.heightPx <= 0) continue;

    // Oversized blocks can span multiple pages; keep them offset-free to avoid
    // representing ambiguous multi-break positions in a single offset field.
    if (block.heightPx > writableHeightPx) continue;

    const nextBoundaryPx =
      (Math.floor(block.topPx / writableHeightPx) + 1) * writableHeightPx;
    if (block.bottomPx <= nextBoundaryPx) continue;

    const breakOffsetPx = nextBoundaryPx - block.topPx;
    if (breakOffsetPx <= 0 || breakOffsetPx >= block.heightPx) continue;
    if (breakOffsetsByBlockId[block.id] !== undefined) continue;

    breakOffsetsByBlockId[block.id] = breakOffsetPx;
  }

  const maxContentBottomPx = blocks.reduce(
    (maxBottomPx, block) => Math.max(maxBottomPx, block.bottomPx),
    0,
  );
  const pageCount = Math.max(1, Math.ceil(maxContentBottomPx / writableHeightPx));

  return {
    pageCount,
    breakOffsetsByBlockId,
  };
}
