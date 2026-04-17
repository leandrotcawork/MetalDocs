import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Widget } from 'ckeditor5';
import { registerFieldSchema } from '../schema';
import { registerFieldConverters } from '../converters';

describe('field converters', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Widget],
    });
    registerFieldSchema(editor.model.schema);
    registerFieldConverters(editor);
  });

  afterEach(async () => {
    await editor.destroy();
  });

  const sampleHtml =
    '<p><span class="mddm-field" data-field-id="customer" data-field-type="text" data-field-label="Customer" data-field-required="true">Acme</span></p>';

  it('upcasts a span.mddm-field into mddmField model element', () => {
    editor.setData(sampleHtml);
    const root = editor.model.document.getRoot()!;
    const para = root.getChild(0);
    const field = para!.getChild(0) as { name: string; getAttribute(k: string): unknown };
    expect(field.name).toBe('mddmField');
    expect(field.getAttribute('fieldId')).toBe('customer');
    expect(field.getAttribute('fieldType')).toBe('text');
    expect(field.getAttribute('fieldLabel')).toBe('Customer');
    expect(field.getAttribute('fieldRequired')).toBe(true);
    expect(field.getAttribute('fieldValue')).toBe('Acme');
  });

  it('round-trips HTML via setData/getData', () => {
    editor.setData(sampleHtml);
    const out = editor.getData();
    expect(out).toContain('class="mddm-field"');
    expect(out).toContain('data-field-id="customer"');
    expect(out).toContain('data-field-type="text"');
    expect(out).toContain('data-field-label="Customer"');
    expect(out).toContain('data-field-required="true"');
    expect(out).toContain('>Acme<');
  });
});
