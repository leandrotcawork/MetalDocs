import type { Editor, DowncastConversionApi, UpcastConversionApi } from 'ckeditor5';
import { toWidget, toWidgetEditable } from 'ckeditor5';

export function registerSectionConverters(editor: Editor): void {
  const c = editor.conversion;

  // Upcast
  c.for('upcast').elementToElement({
    view: { name: 'section', classes: 'mddm-section' },
    model: (viewEl, { writer }: UpcastConversionApi) =>
      writer.createElement('mddmSection', {
        sectionId: viewEl.getAttribute('data-section-id') ?? undefined,
        variant: viewEl.getAttribute('data-variant') ?? 'editable',
      }),
  });
  c.for('upcast').elementToElement({
    view: { name: 'header', classes: 'mddm-section__header' },
    model: 'mddmSectionHeader',
  });
  c.for('upcast').elementToElement({
    view: { name: 'div', classes: 'mddm-section__body' },
    model: 'mddmSectionBody',
  });

  // Data downcast — plain HTML for persistence
  c.for('dataDowncast').elementToElement({
    model: 'mddmSection',
    view: (modelEl, { writer }: DowncastConversionApi) =>
      writer.createContainerElement('section', {
        class: 'mddm-section',
        'data-section-id': String(modelEl.getAttribute('sectionId') ?? ''),
        'data-variant': String(modelEl.getAttribute('variant') ?? 'editable'),
      }),
  });
  c.for('dataDowncast').elementToElement({
    model: 'mddmSectionHeader',
    view: (_m, { writer }) => writer.createContainerElement('header', { class: 'mddm-section__header' }),
  });
  c.for('dataDowncast').elementToElement({
    model: 'mddmSectionBody',
    view: (_m, { writer }) => writer.createContainerElement('div', { class: 'mddm-section__body' }),
  });

  // Editing downcast — wraps chrome with widget helpers
  c.for('editingDowncast').elementToElement({
    model: 'mddmSection',
    view: (modelEl, { writer }: DowncastConversionApi) => {
      const section = writer.createContainerElement('section', {
        class: 'mddm-section',
        'data-section-id': String(modelEl.getAttribute('sectionId') ?? ''),
        'data-variant': String(modelEl.getAttribute('variant') ?? 'editable'),
      });
      return toWidget(section, writer, { label: 'section widget' });
    },
  });
  c.for('editingDowncast').elementToElement({
    model: 'mddmSectionHeader',
    view: (_m, { writer }) => {
      const header = writer.createEditableElement('header', { class: 'mddm-section__header' });
      return toWidgetEditable(header, writer);
    },
  });
  c.for('editingDowncast').elementToElement({
    model: 'mddmSectionBody',
    view: (_m, { writer }) => {
      const body = writer.createEditableElement('div', { class: 'mddm-section__body' });
      return toWidgetEditable(body, writer);
    },
  });
}
