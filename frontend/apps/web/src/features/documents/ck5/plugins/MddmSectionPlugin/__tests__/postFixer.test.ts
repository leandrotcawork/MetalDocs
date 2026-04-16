import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Widget } from 'ckeditor5';
import type { ModelElement } from 'ckeditor5';
import { registerSectionSchema } from '../schema';
import { registerSectionPostFixer } from '../postFixer';
import { registerSectionConverters } from '../converters';

describe('section post-fixer', () => {
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
    // Converters required: when postFixer auto-inserts header/body/paragraph,
    // CK5 immediately downcasts them — without converters the engine throws
    // mapping-model-position-view-parent-not-found.
    registerSectionConverters(editor);
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('auto-creates header+body when a bare section is inserted', () => {
    editor.model.change((writer) => {
      const section = writer.createElement('mddmSection');
      writer.append(section, editor.model.document.getRoot()!);
    });
    // Root starts with an initial paragraph; the appended section is after it.
    const root = editor.model.document.getRoot()!;
    const allChildren = Array.from(root.getChildren()) as ModelElement[];
    const section = allChildren.find((c) => c.name === 'mddmSection');
    expect(section).toBeDefined();
    const children = Array.from((section as unknown as { getChildren(): Iterable<{ name: string }> }).getChildren());
    expect(children.map((c) => c.name)).toEqual(['mddmSectionHeader', 'mddmSectionBody']);
  });
});
