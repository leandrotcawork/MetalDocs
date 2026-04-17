import type { Editor, DowncastConversionApi, UpcastConversionApi } from 'ckeditor5';
import { toWidget } from 'ckeditor5';

export function registerFieldConverters(editor: Editor): void {
  const conversion = editor.conversion;

  conversion.for('upcast').elementToElement({
    view: { name: 'span', classes: 'mddm-field' },
    model: (viewEl, { writer }: UpcastConversionApi) =>
      writer.createElement('mddmField', {
        fieldId: viewEl.getAttribute('data-field-id') ?? '',
        fieldType: viewEl.getAttribute('data-field-type') ?? 'text',
        fieldLabel: viewEl.getAttribute('data-field-label') ?? '',
        fieldRequired: viewEl.getAttribute('data-field-required') === 'true',
        fieldValue: viewEl.getChild(0)?.is('$text') ? (viewEl.getChild(0) as { data: string }).data : '',
      }),
  });

  conversion.for('dataDowncast').elementToElement({
    model: 'mddmField',
    view: (modelEl, { writer }: DowncastConversionApi) => {
      const span = writer.createContainerElement('span', {
        class: 'mddm-field',
        'data-field-id': String(modelEl.getAttribute('fieldId') ?? ''),
        'data-field-type': String(modelEl.getAttribute('fieldType') ?? 'text'),
        'data-field-label': String(modelEl.getAttribute('fieldLabel') ?? ''),
        'data-field-required': String(!!modelEl.getAttribute('fieldRequired')),
      });
      const value = String(modelEl.getAttribute('fieldValue') ?? '');
      writer.insert(writer.createPositionAt(span, 0), writer.createText(value));
      return span;
    },
  });

  conversion.for('editingDowncast').elementToElement({
    model: 'mddmField',
    view: (modelEl, { writer }: DowncastConversionApi) => {
      const type = String(modelEl.getAttribute('fieldType') ?? 'text');
      const label = String(modelEl.getAttribute('fieldLabel') ?? '');
      const id = String(modelEl.getAttribute('fieldId') ?? '');
      const value = String(modelEl.getAttribute('fieldValue') ?? '');
      const typeFamily = type.split(':')[0];

      const chip = writer.createContainerElement('span', {
        class: `mddm-field mddm-field--${typeFamily}`,
        'data-field-id': id,
        'aria-label': `${label} (${type})`,
        role: 'textbox',
      });
      writer.insert(writer.createPositionAt(chip, 0), writer.createText(value || `{{${label || id}}}`));
      return toWidget(chip, writer, { label: `${label || id} field` });
    },
  });
}
