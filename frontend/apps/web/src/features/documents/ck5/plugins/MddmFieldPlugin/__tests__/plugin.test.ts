import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Widget } from 'ckeditor5';
import { MddmFieldPlugin } from '../index';

describe('MddmFieldPlugin', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Widget, MddmFieldPlugin],
    });
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('registers the insertMddmField command', () => {
    expect(editor.commands.get('insertMddmField')).toBeDefined();
  });

  it('round-trips field HTML without loss', () => {
    const input =
      '<p><span class="mddm-field" data-field-id="x" data-field-type="text" data-field-label="L" data-field-required="false">v</span></p>';
    editor.setData(input);
    const out = editor.getData();
    expect(out).toContain('data-field-id="x"');
    expect(out).toContain('>v<');
  });
});
