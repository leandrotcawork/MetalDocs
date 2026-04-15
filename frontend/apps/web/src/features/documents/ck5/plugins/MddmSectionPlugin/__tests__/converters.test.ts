import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Widget } from 'ckeditor5';
import { registerSectionSchema } from '../schema';
import { registerSectionPostFixer } from '../postFixer';
import { registerSectionConverters } from '../converters';

describe('section converters', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Widget],
    });
    registerSectionSchema(editor.model.schema);
    registerSectionPostFixer(editor);
    registerSectionConverters(editor);
  });

  afterEach(async () => {
    await editor.destroy();
  });

  const sampleHtml = [
    '<section class="mddm-section" data-section-id="s1" data-variant="mixed">',
    '<header class="mddm-section__header"><p>Title</p></header>',
    '<div class="mddm-section__body"><p>Body</p></div>',
    '</section>',
  ].join('');

  it('round-trips section HTML', () => {
    editor.setData(sampleHtml);
    const out = editor.getData();
    expect(out).toContain('class="mddm-section"');
    expect(out).toContain('data-section-id="s1"');
    expect(out).toContain('data-variant="mixed"');
    expect(out).toContain('class="mddm-section__header"');
    expect(out).toContain('class="mddm-section__body"');
    expect(out).toContain('>Title<');
    expect(out).toContain('>Body<');
  });

  it('defaults variant to editable when missing', () => {
    editor.setData(
      '<section class="mddm-section"><header class="mddm-section__header"/><div class="mddm-section__body"><p/></div></section>',
    );
    const section = editor.model.document.getRoot()!.getChild(0);
    expect((section as { getAttribute(k: string): unknown }).getAttribute('variant')).toBe('editable');
  });
});
