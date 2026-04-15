import type { Schema } from 'ckeditor5';

export function registerFieldSchema(schema: Schema): void {
  schema.register('mddmField', {
    inheritAllFrom: '$inlineObject',
    allowAttributes: ['fieldId', 'fieldType', 'fieldLabel', 'fieldRequired', 'fieldValue'],
  });
}
