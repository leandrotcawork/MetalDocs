import type { Editor } from 'ckeditor5';

export const PAGINABLE_ELEMENT_NAMES = [
  'paragraph',
  'heading1', 'heading2', 'heading3', 'heading4', 'heading5', 'heading6',
  'listItem',
  'blockQuote',
  'tableRow',
  'imageBlock',
  'mediaEmbed',
  'mddmSection',
  'mddmRepeatable',
  'mddmRepeatableItem',
  'mddmDataTable',
  'mddmFieldGroup',
  'mddmRichBlock',
] as const;

export function extendSchemaWithBid(editor: Editor): void {
  const schema = editor.model.schema;
  for (const name of PAGINABLE_ELEMENT_NAMES) {
    if (schema.isRegistered(name)) {
      schema.extend(name, { allowAttributes: ['mddmBid'] });
    }
  }
}
