import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';

import { registerRepeatableSchema } from '../schema';

describe('registerRepeatableSchema', () => {
  let editor: ClassicEditor;
  let element: HTMLElement;

  beforeEach(async () => {
    element = document.createElement('div');
    document.body.appendChild(element);

    editor = await ClassicEditor.create(element, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph],
    });

    registerRepeatableSchema(editor.model.schema);
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('registers mddmRepeatable and mddmRepeatableItem', () => {
    expect(editor.model.schema.isRegistered('mddmRepeatable')).toBe(true);
    expect(editor.model.schema.isRegistered('mddmRepeatableItem')).toBe(true);
  });

  it('allows repeatableId label min max numberingStyle attributes on mddmRepeatable', () => {
    const schema = editor.model.schema;

    expect(schema.checkAttribute('mddmRepeatable', 'repeatableId')).toBe(true);
    expect(schema.checkAttribute('mddmRepeatable', 'label')).toBe(true);
    expect(schema.checkAttribute('mddmRepeatable', 'min')).toBe(true);
    expect(schema.checkAttribute('mddmRepeatable', 'max')).toBe(true);
    expect(schema.checkAttribute('mddmRepeatable', 'numberingStyle')).toBe(true);
  });

  it('only allows mddmRepeatableItem inside mddmRepeatable (not paragraph)', () => {
    const schema = editor.model.schema;

    expect(schema.checkChild(['$root', 'mddmRepeatable'], 'mddmRepeatableItem')).toBe(true);
    expect(schema.checkChild(['$root', 'paragraph'], 'mddmRepeatableItem')).toBe(false);
  });

  it('forbids nested mddmRepeatable inside mddmRepeatableItem', () => {
    const schema = editor.model.schema;

    expect(
      schema.checkChild(['$root', 'mddmRepeatable', 'mddmRepeatableItem'], 'mddmRepeatable'),
    ).toBe(false);
  });
});
