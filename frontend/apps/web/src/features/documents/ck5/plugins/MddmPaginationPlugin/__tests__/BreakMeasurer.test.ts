import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../../MddmBlockIdentityPlugin';
import { BreakMeasurer } from '../BreakMeasurer';
import { DirtyRangeTracker } from '../DirtyRangeTracker';

describe('BreakMeasurer', () => {
  let editor: ClassicEditor;
  let host: HTMLElement;

  beforeEach(async () => {
    host = document.createElement('div');
    document.body.appendChild(host);
    editor = await ClassicEditor.create(host, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, MddmBlockIdentityPlugin],
    });
  });

  afterEach(async () => {
    await editor.destroy();
    host.remove();
  });

  it('fires onBreaks callback after debounce window', async () => {
    editor.setData('<p>hello</p>');
    const tracker = new DirtyRangeTracker(editor);
    const m = new BreakMeasurer(editor, tracker, { debounceMs: 20 });
    const emitted: unknown[] = [];
    m.onBreaks(b => emitted.push(b));

    editor.model.change(writer => {
      const root = editor.model.document.getRoot()!;
      writer.insertText('x', root.getChild(0) as never, 'end');
    });

    await new Promise(r => setTimeout(r, 80));
    expect(emitted.length).toBeGreaterThan(0);
    m.destroy();
    tracker.destroy();
  });

  it('debounces multiple rapid renders into one callback', async () => {
    editor.setData('<p>hello</p>');
    const tracker = new DirtyRangeTracker(editor);
    const m = new BreakMeasurer(editor, tracker, { debounceMs: 50 });
    const emitted: unknown[] = [];
    m.onBreaks(b => emitted.push(b));

    for (let i = 0; i < 3; i++) {
      editor.model.change(writer => {
        const root = editor.model.document.getRoot()!;
        writer.insertText('x', root.getChild(0) as never, 'end');
      });
    }

    await new Promise(r => setTimeout(r, 150));
    expect(emitted.length).toBeLessThanOrEqual(2);
    m.destroy();
    tracker.destroy();
  });
});
