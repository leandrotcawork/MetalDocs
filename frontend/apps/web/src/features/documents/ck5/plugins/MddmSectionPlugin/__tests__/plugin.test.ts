import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Widget, StandardEditingMode } from 'ckeditor5';
import { MddmSectionPlugin } from '../index';

describe('MddmSectionPlugin integration', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Widget, StandardEditingMode, MddmSectionPlugin],
    });
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('registers insertMddmSection', () => {
    expect(editor.commands.get('insertMddmSection')).toBeDefined();
  });
});
