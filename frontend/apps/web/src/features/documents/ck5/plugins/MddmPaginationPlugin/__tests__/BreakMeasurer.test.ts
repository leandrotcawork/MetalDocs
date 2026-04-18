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

  it('emits preventive break when next candidate would overflow current page', async () => {
    editor.setData(
      '<p data-mddm-bid="a">A</p><p data-mddm-bid="b">B</p><p data-mddm-bid="c">C</p>',
    );

    const root = editor.editing.view.getDomRoot() as HTMLElement;
    const fakes: Array<[string, number, number]> = [
      ['a', 94.49, 100],
      ['b', 194.49, 900],
      ['c', 1094.49, 50],
    ];
    for (const [bid, top, height] of fakes) {
      const el = root.querySelector(`[data-mddm-bid="${bid}"]`) as HTMLElement | null;
      if (!el) throw new Error(`missing bid ${bid}`);
      Object.defineProperty(el, 'offsetTop', { configurable: true, get: () => top });
      Object.defineProperty(el, 'offsetHeight', { configurable: true, get: () => height });
    }

    const tracker = new DirtyRangeTracker(editor);
    const measurer = new BreakMeasurer(editor, tracker, { debounceMs: 10 });
    const emitted: Array<{ afterBid: string; pageNumber: number; spacerPx: number }> = [];
    measurer.onBreaks(batch => emitted.push(...batch));

    editor.model.change(writer => {
      const r = editor.model.document.getRoot()!;
      writer.insertText(' ', r.getChild(0) as never, 'end');
    });
    await new Promise(r => setTimeout(r, 40));

    expect(emitted.length).toBeGreaterThan(0);
    expect(emitted[emitted.length - 1]).toMatchObject({ afterBid: 'a', pageNumber: 2 });
    expect(emitted[emitted.length - 1].spacerPx).toBeCloseTo(1054.52, 0);

    measurer.destroy();
    tracker.destroy();
  });
});
