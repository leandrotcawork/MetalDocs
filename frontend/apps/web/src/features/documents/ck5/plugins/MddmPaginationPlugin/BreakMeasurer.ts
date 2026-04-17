import type { Editor } from 'ckeditor5';
import { defaultLayoutTokens } from '@metaldocs/mddm-layout-tokens';
import { planBreaks } from './BreakPlanner';
import type { DirtyRangeTracker } from './DirtyRangeTracker';
import type { ComputedBreak } from './types';

const MM_PER_INCH = 25.4;
const PX_PER_INCH = 96;
const mmToPx = (mm: number) => (mm / MM_PER_INCH) * PX_PER_INCH;

type Listener = (breaks: ComputedBreak[]) => void;

/**
 * Subscribes to view `render` events, debounces, then measures DOM heights
 * for each BreakCandidate and emits ComputedBreak[] via `onBreaks` listeners.
 */
export class BreakMeasurer {
  private readonly listeners = new Set<Listener>();
  private timer: ReturnType<typeof setTimeout> | null = null;
  private readonly renderHandler: () => void;
  private fontsReady = false;

  public constructor(
    private readonly editor: Editor,
    private readonly tracker: DirtyRangeTracker,
    private readonly opts: { debounceMs: number } = { debounceMs: 200 },
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

    const dirty = this.tracker.snapshot();
    const from = dirty ?? this.editor.model.createPositionFromPath(root, [0]);
    const candidates = planBreaks(this.editor, from);

    const dpr = (typeof window !== 'undefined' && window.devicePixelRatio) || 1;
    const contentHeightPx = mmToPx(
      defaultLayoutTokens.page.heightMm -
        defaultLayoutTokens.page.marginTopMm -
        defaultLayoutTokens.page.marginBottomMm,
    );

    const breaks: ComputedBreak[] = [];
    let cursorY = 0;
    let page = 1;

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

      const h = Math.round(domEl.offsetHeight * dpr) / dpr;
      cursorY += h;

      if (cursorY > contentHeightPx) {
        page += 1;
        breaks.push({ afterBid: c.afterBid, pageNumber: page, yPx: cursorY });
        cursorY = h;
      }
    }

    for (const l of this.listeners) l(breaks);
  }
}
