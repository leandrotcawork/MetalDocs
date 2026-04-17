import type { Editor, Position as ModelPosition } from 'ckeditor5';

/**
 * Tracks the earliest model position modified since the last snapshot.
 *
 * Subscribes to `editor.model.document` `change:data` events, reads
 * `Differ.getChanges()`, and records the earliest affected position.
 * Multiple rapid edits collapse into a single position (the earliest).
 */
export class DirtyRangeTracker {
  private dirtyStart: ModelPosition | null = null;
  private readonly handler: () => void;

  public constructor(private readonly editor: Editor) {
    this.handler = () => this.onChange();
    this.editor.model.document.on('change:data', this.handler);
  }

  public destroy(): void {
    this.editor.model.document.off('change:data', this.handler);
  }

  /** Returns the earliest dirty position since the last snapshot and clears state. */
  public snapshot(): ModelPosition | null {
    const out = this.dirtyStart;
    this.dirtyStart = null;
    return out;
  }

  private onChange(): void {
    const changes = this.editor.model.document.differ.getChanges();
    for (const change of changes) {
      const pos = (change as { position?: ModelPosition }).position;
      if (!pos) continue;
      if (!this.dirtyStart || pos.isBefore(this.dirtyStart)) {
        this.dirtyStart = pos;
      }
    }
  }
}
