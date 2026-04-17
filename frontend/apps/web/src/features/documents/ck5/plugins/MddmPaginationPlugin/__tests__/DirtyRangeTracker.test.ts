import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../../MddmBlockIdentityPlugin';
import { DirtyRangeTracker } from '../DirtyRangeTracker';

describe('DirtyRangeTracker', () => {
  let editor: ClassicEditor;
  let tracker: DirtyRangeTracker;
  let host: HTMLElement;

  beforeEach(async () => {
    host = document.createElement('div');
    document.body.appendChild(host);
    editor = await ClassicEditor.create(host, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, MddmBlockIdentityPlugin],
    });
    editor.setData('<p>A</p><p>B</p><p>C</p>');
    tracker = new DirtyRangeTracker(editor);
  });

  afterEach(async () => {
    tracker.destroy();
    await editor.destroy();
    host.remove();
  });

  it('starts with null snapshot', () => {
    expect(tracker.snapshot()).toBeNull();
  });

  it('records position of edit and resets on snapshot', () => {
    editor.model.change(writer => {
      const p = editor.model.document.getRoot()!.getChild(0)!;
      writer.insertText('x', p as never, 'end');
    });
    expect(tracker.snapshot()).not.toBeNull();
    expect(tracker.snapshot()).toBeNull();
  });

  it('collapses two edits — earlier position wins', () => {
    editor.model.change(writer => {
      const p = editor.model.document.getRoot()!.getChild(2)!;
      writer.insertText('x', p as never, 'end');
    });
    const s1 = tracker.snapshot();
    expect(s1).not.toBeNull();

    editor.model.change(writer => {
      const p = editor.model.document.getRoot()!.getChild(0)!;
      writer.insertText('y', p as never, 'end');
    });
    const s2 = tracker.snapshot();
    // p[0] is before p[2], so s2 should be before s1
    expect(s2!.isBefore(s1!) || s2!.isEqual(s1!)).toBe(true);
  });
});
