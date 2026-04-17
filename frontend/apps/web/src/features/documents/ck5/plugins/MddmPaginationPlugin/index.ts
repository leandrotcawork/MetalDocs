import { Plugin } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../MddmBlockIdentityPlugin';
import { DirtyRangeTracker } from './DirtyRangeTracker';
import { BreakMeasurer } from './BreakMeasurer';

export class MddmPaginationPlugin extends Plugin {
  public static get pluginName() { return 'MddmPagination' as const; }
  public static get requires() { return [MddmBlockIdentityPlugin] as const; }

  public init(): void {
    const tracker = new DirtyRangeTracker(this.editor);
    const measurer = new BreakMeasurer(this.editor, tracker);
    this.on('destroy', () => {
      measurer.destroy();
      tracker.destroy();
    });
    (this as unknown as { _tracker: DirtyRangeTracker })._tracker = tracker;
    (this as unknown as { _measurer: BreakMeasurer })._measurer = measurer;
  }
}
