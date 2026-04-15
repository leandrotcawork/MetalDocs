import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Widget, StandardEditingMode } from 'ckeditor5';
import { registerSectionSchema } from '../schema';
import { registerSectionPostFixer } from '../postFixer';
import { registerSectionConverters } from '../converters';
import { InsertSectionCommand } from '../commands/InsertSectionCommand';

describe('InsertSectionCommand', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Widget, StandardEditingMode],
    });
    registerSectionSchema(editor.model.schema);
    registerSectionPostFixer(editor);
    registerSectionConverters(editor);
    editor.commands.add('insertMddmSection', new InsertSectionCommand(editor));
    // Command.isEnabled starts false; the auto-listener only fires on real model ops.
    // Call refresh() directly to prime isEnabled before the first execute().
    (editor.commands.get('insertMddmSection') as InsertSectionCommand).refresh();
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('inserts a section with variant="editable" by default', () => {
    editor.execute('insertMddmSection');
    const out = editor.getData();
    expect(out).toContain('class="mddm-section"');
    expect(out).toContain('data-variant="editable"');
    expect(out).toContain('class="mddm-section__header"');
    expect(out).toContain('class="mddm-section__body"');
  });

  it('accepts variant parameter', () => {
    editor.execute('insertMddmSection', { variant: 'locked' });
    expect(editor.getData()).toContain('data-variant="locked"');
  });

  it('plants a restricted-editing-exception marker on the body for editable variant', () => {
    editor.execute('insertMddmSection', { variant: 'editable' });
    const markers = Array.from(editor.model.markers).filter((m) =>
      m.name.startsWith('restrictedEditingException:'),
    );
    expect(markers.length).toBeGreaterThanOrEqual(1);
  });
});
