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

  it('renders one overlay per break', () => {
    view.update([
      { afterBid: 'aaa', pageNumber: 2, yPx: 100 },
      { afterBid: 'bbb', pageNumber: 3, yPx: 200 },
    ]);
    expect(document.querySelectorAll('.mddm-page-overlay')).toHaveLength(2);
  });

  it('overlays do not appear in getData()', () => {
    editor.setData('<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>');
    view.update([{ afterBid: 'aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa', pageNumber: 2, yPx: 100 }]);
    expect(editor.getData()).not.toContain('mddm-page-overlay');
  });

  it('update clears previous overlays', () => {
    view.update([{ afterBid: 'aaa', pageNumber: 2, yPx: 100 }]);
    view.update([
      { afterBid: 'bbb', pageNumber: 2, yPx: 100 },
      { afterBid: 'ccc', pageNumber: 3, yPx: 200 },
    ]);
    expect(document.querySelectorAll('.mddm-page-overlay')).toHaveLength(2);
  });
});
