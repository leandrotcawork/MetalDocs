import { Plugin } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../MddmBlockIdentityPlugin';
import { DirtyRangeTracker } from './DirtyRangeTracker';
import { BreakMeasurer } from './BreakMeasurer';
import { PageOverlayView } from './PageOverlayView';
import { installPaginationDataContract } from './data-contract';
import type { ComputedBreak } from './types';

export class MddmPaginationPlugin extends Plugin {
  public static get pluginName() { return 'MddmPagination' as const; }
  public static get requires() { return [MddmBlockIdentityPlugin] as const; }

  public init(): void {
    const tracker = new DirtyRangeTracker(this.editor);
    const measurer = new BreakMeasurer(this.editor, tracker);
    const overlay = new PageOverlayView(this.editor);
    let currentBreaks: readonly ComputedBreak[] = [];
    measurer.onBreaks(b => {
      currentBreaks = b;
      overlay.update(b);
    });
    installPaginationDataContract(this.editor, () => currentBreaks);
    (this as any).setComputedBreaks = (b: readonly ComputedBreak[]) => { currentBreaks = b; };
    this.on('destroy', () => {
      overlay.destroy();
      measurer.destroy();
      tracker.destroy();
    });
    (this as unknown as { _tracker: DirtyRangeTracker })._tracker = tracker;
    (this as unknown as { _measurer: BreakMeasurer })._measurer = measurer;
    (this as unknown as { _overlay: PageOverlayView })._overlay = overlay;
  }
}
