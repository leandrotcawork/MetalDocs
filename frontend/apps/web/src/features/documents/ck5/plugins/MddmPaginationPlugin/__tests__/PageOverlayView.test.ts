import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../../MddmBlockIdentityPlugin';
import { PageOverlayView } from '../PageOverlayView';

describe('PageOverlayView', () => {
  let editor: ClassicEditor;
  let view: PageOverlayView;
  let host: HTMLElement;

  beforeEach(async () => {
    host = document.createElement('div');
    document.body.appendChild(host);
    editor = await ClassicEditor.create(host, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, MddmBlockIdentityPlugin],
    });
    view = new PageOverlayView(editor);
  });

  afterEach(async () => {
    view.destroy();
    await editor.destroy();
    host.remove();
  });

  it('marks one block per break', () => {
    editor.setData([
      '<p data-mddm-bid="aaa">A</p>',
      '<p data-mddm-bid="bbb">B</p>',
    ].join(''));

    view.update([
      { afterBid: 'aaa', pageNumber: 2, yPx: 100 },
      { afterBid: 'bbb', pageNumber: 3, yPx: 200 },
    ]);

    const a = document.querySelector('[data-mddm-bid="aaa"]') as HTMLElement | null;
    const b = document.querySelector('[data-mddm-bid="bbb"]') as HTMLElement | null;

    expect(a?.hasAttribute('data-mddm-page-break-after')).toBe(true);
    expect(a?.getAttribute('data-mddm-next-page')).toBe('2');
    expect(b?.hasAttribute('data-mddm-page-break-after')).toBe(true);
    expect(b?.getAttribute('data-mddm-next-page')).toBe('3');
  });

  it('overlays do not appear in getData()', () => {
    editor.setData('<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>');
    view.update([{ afterBid: 'aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa', pageNumber: 2, yPx: 100 }]);
    expect(editor.getData()).not.toContain('data-mddm-page-break-after');
    expect(editor.getData()).not.toContain('data-mddm-next-page');
  });

  it('update clears previous markers', () => {
    editor.setData([
      '<p data-mddm-bid="aaa">A</p>',
      '<p data-mddm-bid="bbb">B</p>',
      '<p data-mddm-bid="ccc">C</p>',
    ].join(''));

    const a = document.querySelector('[data-mddm-bid="aaa"]') as HTMLElement | null;
    const b = document.querySelector('[data-mddm-bid="bbb"]') as HTMLElement | null;
    const c = document.querySelector('[data-mddm-bid="ccc"]') as HTMLElement | null;

    view.update([{ afterBid: 'aaa', pageNumber: 2, yPx: 100 }]);
    expect(a?.hasAttribute('data-mddm-page-break-after')).toBe(true);
    expect(a?.getAttribute('data-mddm-next-page')).toBe('2');

    view.update([
      { afterBid: 'bbb', pageNumber: 2, yPx: 100 },
      { afterBid: 'ccc', pageNumber: 3, yPx: 200 },
    ]);

    expect(a?.hasAttribute('data-mddm-page-break-after')).toBe(false);
    expect(a?.hasAttribute('data-mddm-next-page')).toBe(false);
    expect(b?.hasAttribute('data-mddm-page-break-after')).toBe(true);
    expect(b?.getAttribute('data-mddm-next-page')).toBe('2');
    expect(c?.hasAttribute('data-mddm-page-break-after')).toBe(true);
    expect(c?.getAttribute('data-mddm-next-page')).toBe('3');
  });
});
