import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Plugin, Widget } from 'ckeditor5';

import { registerRepeatableConverters } from '../converters';
import { registerRepeatableSchema } from '../schema';

class RepeatableConvertersTestPlugin extends Plugin {
  public override init(): void {
    registerRepeatableSchema(this.editor.model.schema);
    registerRepeatableConverters(this.editor);
  }
}

describe('registerRepeatableConverters', () => {
  let editor: ClassicEditor;
  let element: HTMLElement;

  beforeEach(async () => {
    element = document.createElement('div');
    document.body.appendChild(element);

    editor = await ClassicEditor.create(element, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Widget, RepeatableConvertersTestPlugin],
    });
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('round-trips repeatable HTML', async () => {
    const sampleHtml = `
      <ol class="mddm-repeatable" data-repeatable-id="r1" data-label="Items" data-min="1" data-max="5" data-numbering="decimal">
        <li class="mddm-repeatable__item"><p>First</p></li>
        <li class="mddm-repeatable__item"><p>Second</p></li>
      </ol>
    `;

    await editor.setData(sampleHtml);
    const output = editor.getData();

    expect(output).toContain('mddm-repeatable');
    expect(output).toContain('data-repeatable-id="r1"');
    expect(output).toContain('data-min="1"');
    expect(output).toContain('data-max="5"');
    expect(output).toContain('mddm-repeatable__item');
    expect(output).toContain('First');
    expect(output).toContain('Second');
  });
});
