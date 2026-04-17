import { describe, it, expect } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { MddmBlockIdentityPlugin } from '../../MddmBlockIdentityPlugin';
import { MddmPaginationPlugin } from '../index';

describe('MddmPaginationPlugin scaffold', () => {
  it('pluginName is MddmPagination', () => {
    expect(MddmPaginationPlugin.pluginName).toBe('MddmPagination');
  });
  it('requires block identity plugin', () => {
    expect(MddmPaginationPlugin.requires).toContain(MddmBlockIdentityPlugin);
  });
  it('boots inside editor', async () => {
    const host = document.createElement('div');
    document.body.appendChild(host);
    const editor = await ClassicEditor.create(host, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, MddmBlockIdentityPlugin, MddmPaginationPlugin],
    });
    expect(editor.plugins.has('MddmPagination')).toBe(true);
    await editor.destroy();
    host.remove();
  });
});
