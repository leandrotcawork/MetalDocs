import type { Editor } from 'ckeditor5';
import { defaultLayoutTokens } from '@metaldocs/mddm-layout-tokens';
import { planBreaks } from './BreakPlanner';
import type { DirtyRangeTracker } from './DirtyRangeTracker';
import type { ComputedBreak } from './types';

const MM_PER_INCH = 25.4;
const PX_PER_INCH = 96;
const mmToPx = (mm: number) => (mm / MM_PER_INCH) * PX_PER_INCH;
const PAGE_HEIGHT_PX = mmToPx(defaultLayoutTokens.page.heightMm);
const PAGE_GAP_PX = 32;
const STRIDE_PX = PAGE_HEIGHT_PX + PAGE_GAP_PX;
const MARGIN_TOP_PX = mmToPx(defaultLayoutTokens.page.marginTopMm);

type Listener = (breaks: ComputedBreak[]) => void;

export class BreakMeasurer {
  private readonly listeners = new Set<Listener>();
  private timer: ReturnType<typeof setTimeout> | null = null;
  private readonly renderHandler: () => void;
  private fontsReady = false;
  private initialSyncDone = false;

  public constructor(
    private readonly editor: Editor,
    private readonly tracker: DirtyRangeTracker,
    private readonly opts: { debounceMs: number } = { debounceMs: 50 },
  ) {
    this.renderHandler = () => this.schedule();
    (this.editor.editing.view as unknown as { on: (n: string, f: () => void) => void }).on(
      'render',
      this.renderHandler,
    );
  }

  public destroy(): void {
    (this.editor.editing.view as unknown as { off: (n: string, f: () => void) => void }).off(
      'render',
      this.renderHandler,
    );
    if (this.timer !== null) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    this.listeners.clear();
  }

  public onBreaks(fn: Listener): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  private schedule(): void {
    if (!this.initialSyncDone) {
      this.initialSyncDone = true;
      void this.measure();
      return;
    }
    if (this.timer !== null) clearTimeout(this.timer);
    this.timer = setTimeout(() => {
      this.timer = null;
      void this.measure();
    }, this.opts.debounceMs);
  }

  private async measure(): Promise<void> {
    if (!this.fontsReady) {
      try {
        if (typeof document !== 'undefined' && document.fonts?.ready) {
          await document.fonts.ready;
        }
      } catch {
        /* jsdom — no fonts API */
      }
      this.fontsReady = true;
    }

    const root = this.editor.model.document.getRoot();
    if (!root) return;

    const from = this.editor.model.createPositionFromPath(root, [0]);
    void this.tracker.snapshot();
    const candidates = planBreaks(this.editor, from, undefined);

    const dpr = (typeof window !== 'undefined' && window.devicePixelRatio) || 1;
    const contentHeightPx = mmToPx(
      defaultLayoutTokens.page.heightMm -
        defaultLayoutTokens.page.marginTopMm -
        defaultLayoutTokens.page.marginBottomMm,
    );

    const page1TopY = MARGIN_TOP_PX;

    const breaks: ComputedBreak[] = [];
    let currentPage = 1;
    let prevAfterBid: string | null = null;
    let prevCandidateBot = 0;

    for (const c of candidates) {
      const pos = this.editor.model.createPositionFromPath(root, c.modelPath);
      const modelEl = pos.nodeBefore;
      if (!modelEl || !modelEl.is('element')) continue;

      const viewEl = this.editor.editing.mapper.toViewElement(modelEl);
      if (!viewEl) continue;
      const domEl = this.editor.editing.view.domConverter.mapViewToDom(viewEl) as
        | HTMLElement
        | undefined;
      if (!domEl) continue;

      const imgs = domEl.querySelectorAll('img');
      for (const img of Array.from(imgs)) {
        try {
          await (img as HTMLImageElement).decode();
        } catch {
          console.warn('mddm:pagination-measure-skip', c.afterBid);
        }
      }

      const top = domEl.offsetTop;
      const bot = Math.round((top + domEl.offsetHeight) * dpr) / dpr;

      const predictedPageTop = page1TopY + (currentPage - 1) * STRIDE_PX;
      const predictedPageCap = predictedPageTop + contentHeightPx;

      if (bot > predictedPageCap && prevAfterBid !== null) {
        currentPage += 1;
        const targetNextTop = page1TopY + (currentPage - 1) * STRIDE_PX;
        const spacerPx = Math.max(0, targetNextTop - prevCandidateBot);
        breaks.push({
          afterBid: prevAfterBid,
          pageNumber: currentPage,
          yPx: prevCandidateBot,
          spacerPx,
        });
      }

      prevAfterBid = c.afterBid;
      prevCandidateBot = bot;
    }

    for (const l of this.listeners) l(breaks);
  }
}
