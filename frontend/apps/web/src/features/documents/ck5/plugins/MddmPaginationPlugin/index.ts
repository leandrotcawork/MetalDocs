import { Plugin } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../MddmBlockIdentityPlugin';
import { DirtyRangeTracker } from './DirtyRangeTracker';
import { BreakMeasurer } from './BreakMeasurer';
import { PageOverlayView } from './PageOverlayView';

export class MddmPaginationPlugin extends Plugin {
  public static get pluginName() { return 'MddmPagination' as const; }
  public static get requires() { return [MddmBlockIdentityPlugin] as const; }

  public init(): void {
    const tracker = new DirtyRangeTracker(this.editor);
    const measurer = new BreakMeasurer(this.editor, tracker);
    const overlay = new PageOverlayView(this.editor);
    measurer.onBreaks((breaks) => overlay.update(breaks));
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
