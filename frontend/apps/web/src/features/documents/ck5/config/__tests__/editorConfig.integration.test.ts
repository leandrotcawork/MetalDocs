import { describe, it, expect } from 'vitest';
import { ClassicEditor } from 'ckeditor5';
import { createAuthorConfig, createFillConfig } from '../editorConfig';

describe('editorConfig integration', () => {
  it('Author editor exposes all primitive-insertion commands', async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    const editor = await ClassicEditor.create(el, createAuthorConfig({}));
    expect(editor.commands.get('insertMddmField')).toBeDefined();
    expect(editor.commands.get('insertMddmSection')).toBeDefined();
    expect(editor.commands.get('insertMddmRepeatable')).toBeDefined();
    expect(editor.commands.get('insertMddmRichBlock')).toBeDefined();
    await editor.destroy();
  });

  it('Fill editor loads primitive schemas so template HTML upcasts correctly', async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    const editor = await ClassicEditor.create(el, createFillConfig({}));
    expect(editor.model.schema.getDefinition('mddmField')).toBeDefined();
    expect(editor.model.schema.getDefinition('mddmSection')).toBeDefined();
    expect(editor.model.schema.getDefinition('mddmRepeatable')).toBeDefined();
    await editor.destroy();
  });

  it('Fill editor round-trips template HTML with section + field without data loss', async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    const editor = await ClassicEditor.create(el, createFillConfig({}));
    const html = [
      '<section class="mddm-section" data-section-id="s1" data-variant="editable">',
      '<header class="mddm-section__header"><p>T</p></header>',
      '<div class="mddm-section__body"><p>',
      '<span class="mddm-field" data-field-id="x" data-field-type="text" data-field-label="X" data-field-required="false">v</span>',
      '</p></div></section>',
    ].join('');
    editor.setData(html);
    const out = editor.getData();
    expect(out).toContain('class="mddm-section"');
    expect(out).toContain('data-field-id="x"');
    expect(out).toContain('>v<');
    await editor.destroy();
  });
});
