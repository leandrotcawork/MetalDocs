import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { registerFieldSchema } from '../schema';

describe('field schema', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph],
    });
    registerFieldSchema(editor.model.schema);
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('registers mddmField as an inline object', () => {
    const def = editor.model.schema.getDefinition('mddmField');
    expect(def).toBeDefined();
    expect(def!.isObject).toBe(true);
    expect(def!.isInline).toBe(true);
  });

  it('allows fieldId, fieldType, fieldLabel, fieldRequired, fieldValue attributes', () => {
    const attrs = ['fieldId', 'fieldType', 'fieldLabel', 'fieldRequired', 'fieldValue'];
    for (const attr of attrs) {
      expect(editor.model.schema.checkAttribute(['$root', 'paragraph', 'mddmField'], attr)).toBe(true);
    }
  });

  it('is allowed inside a paragraph', () => {
    expect(editor.model.schema.checkChild(['$root', 'paragraph'], 'mddmField')).toBe(true);
  });
});
