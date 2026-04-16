import type { Editor, ModelElement, ViewElement } from 'ckeditor5';
import { toWidget, toWidgetEditable } from 'ckeditor5';

function parseNumberAttribute(value: unknown, fallback: number): number {
  if (value === undefined || value === null || value === '') {
    return fallback;
  }

  const parsed = Number.parseInt(String(value), 10);
  return Number.isNaN(parsed) ? fallback : parsed;
}

export function registerRepeatableConverters(editor: Editor): void {
  const conversion = editor.conversion;

  conversion.for('upcast').elementToElement({
    view: {
      name: 'ol',
      classes: 'mddm-repeatable',
    },
    model: (viewElement, { writer }) => {
      const repeatableId = viewElement.getAttribute('data-repeatable-id') ?? '';
      const label = viewElement.getAttribute('data-label') ?? '';
      const min = parseNumberAttribute(viewElement.getAttribute('data-min'), 1);
      const rawMax = viewElement.getAttribute('data-max');
      const max = rawMax === '' || rawMax === null ? Infinity : parseNumberAttribute(rawMax, Infinity);
      const numberingStyle = viewElement.getAttribute('data-numbering') ?? '';

      return writer.createElement('mddmRepeatable', {
        repeatableId,
        label,
        min,
        max,
        numberingStyle,
      });
    },
  });

  conversion.for('upcast').elementToElement({
    view: {
      name: 'li',
      classes: 'mddm-repeatable__item',
    },
    model: 'mddmRepeatableItem',
  });

  conversion.for('dataDowncast').elementToElement({
    model: 'mddmRepeatable',
    view: (modelElement, { writer }) => {
      const max = modelElement.getAttribute('max');
      return writer.createContainerElement('ol', {
        class: 'mddm-repeatable',
        'data-repeatable-id': String(modelElement.getAttribute('repeatableId') ?? ''),
        'data-label': String(modelElement.getAttribute('label') ?? ''),
        'data-min': String(modelElement.getAttribute('min') ?? ''),
        'data-max': max === Infinity ? '' : String(max ?? ''),
        'data-numbering': String(modelElement.getAttribute('numberingStyle') ?? ''),
      });
    },
  });

  conversion.for('dataDowncast').elementToElement({
    model: 'mddmRepeatableItem',
    view: (modelElement: ModelElement, { writer }) =>
      writer.createContainerElement('li', { class: 'mddm-repeatable__item' }),
  });

  conversion.for('editingDowncast').elementToElement({
    model: 'mddmRepeatable',
    view: (modelElement: ModelElement, { writer }) => {
      const element = writer.createContainerElement('ol', {
        class: 'mddm-repeatable',
      });
      return toWidget(element, writer);
    },
  });

  conversion.for('editingDowncast').elementToElement({
    model: 'mddmRepeatableItem',
    view: (modelElement: ModelElement, { writer }) => {
      const element = writer.createEditableElement('li', { class: 'mddm-repeatable__item' });
      return toWidgetEditable(element, writer);
    },
  });
}
