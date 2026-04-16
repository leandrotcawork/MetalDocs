import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Widget } from 'ckeditor5';
import { registerFieldSchema } from '../schema';
import { registerFieldConverters } from '../converters';
import { InsertFieldCommand } from '../commands/InsertFieldCommand';

describe('InsertFieldCommand', () => {
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
    editor.commands.add('insertMddmField', new InsertFieldCommand(editor));
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('inserts a field at selection', () => {
    editor.setData('<p>Hello </p>');
    editor.model.change((writer) => {
      const root = editor.model.document.getRoot()!;
      const para = root.getChild(0)!;
      writer.setSelection(writer.createPositionAt(para, 'end'));
    });
    editor.execute('insertMddmField', {
      fieldId: 'name',
      fieldType: 'text',
      fieldLabel: 'Name',
      fieldRequired: true,
      fieldValue: '',
    });
    const html = editor.getData();
    expect(html).toContain('data-field-id="name"');
    expect(html).toContain('data-field-type="text"');
    expect(html).toContain('data-field-required="true"');
  });

  it('is enabled inside a paragraph, disabled at the root', () => {
    editor.setData('<p>x</p>');
    const cmd = editor.commands.get('insertMddmField')!;
    editor.model.change((writer) => {
      const para = editor.model.document.getRoot()!.getChild(0)!;
      writer.setSelection(writer.createPositionAt(para, 0));
    });
    expect(cmd.isEnabled).toBe(true);
  });
});
