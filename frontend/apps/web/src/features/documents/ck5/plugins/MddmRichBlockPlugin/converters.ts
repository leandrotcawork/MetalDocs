import type { Editor } from 'ckeditor5';
import { toWidgetEditable } from 'ckeditor5';

export function registerRichBlockConverters(editor: Editor): void {
  const c = editor.conversion;

  c.for('upcast').elementToElement({
    view: { name: 'div', classes: 'mddm-rich-block' },
    model: 'mddmRichBlock',
  });

  c.for('dataDowncast').elementToElement({
    model: 'mddmRichBlock',
    view: (_m, { writer }) =>
      writer.createContainerElement('div', { class: 'mddm-rich-block' }),
  });

  c.for('editingDowncast').elementToElement({
    model: 'mddmRichBlock',
    view: (_m, { writer }) => {
      const div = writer.createEditableElement('div', { class: 'mddm-rich-block' });
      return toWidgetEditable(div, writer);
    },
  });
}
