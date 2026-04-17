import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../../MddmBlockIdentityPlugin';
import { MddmPaginationPlugin } from '../index';

describe('getData pagination option', () => {
  let editor: ClassicEditor;
  let host: HTMLElement;

  beforeEach(async () => {
    host = document.createElement('div');
    document.body.appendChild(host);
    editor = await ClassicEditor.create(host, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, MddmBlockIdentityPlugin, MddmPaginationPlugin],
    });
  });

  afterEach(async () => {
    await editor.destroy();
    host.remove();
  });

  it('no flag -> no pagination attrs', () => {
    editor.setData('<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>');
    expect(editor.getData()).not.toContain('data-pagination-page');
  });

  it('flag set + stub breaks -> attrs present', () => {
    editor.setData('<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">a</p><p data-mddm-bid="bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb">b</p>');
    const plugin = editor.plugins.get('MddmPagination') as MddmPaginationPlugin;
    (plugin as any).setComputedBreaks([{ afterBid: 'aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa', pageNumber: 2, yPx: 100 }]);
    const html = (editor.data as any).get({ pagination: true });
    expect(html).toContain('data-pagination-page="2"');
  });
});
