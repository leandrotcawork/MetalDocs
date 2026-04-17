import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { registerSectionSchema } from '../schema';

describe('section schema', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph],
    });
    registerSectionSchema(editor.model.schema);
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('registers mddmSection, mddmSectionHeader, mddmSectionBody', () => {
    expect(editor.model.schema.getDefinition('mddmSection')).toBeDefined();
    expect(editor.model.schema.getDefinition('mddmSectionHeader')).toBeDefined();
    expect(editor.model.schema.getDefinition('mddmSectionBody')).toBeDefined();
  });

  it('mddmSection is a block object', () => {
    const def = editor.model.schema.getDefinition('mddmSection')!;
    expect(def.isObject).toBe(true);
    expect(def.isBlock).toBe(true);
  });

  it('header and body are limits', () => {
    expect(editor.model.schema.getDefinition('mddmSectionHeader')!.isLimit).toBe(true);
    expect(editor.model.schema.getDefinition('mddmSectionBody')!.isLimit).toBe(true);
  });

  it('allows variant and sectionId attributes on mddmSection', () => {
    const attrs = ['sectionId', 'variant'];
    for (const attr of attrs) {
      expect(editor.model.schema.checkAttribute(['$root', 'mddmSection'], attr)).toBe(true);
    }
  });
});
