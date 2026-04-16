import type { ModelSchema } from 'ckeditor5';

export function registerFieldSchema(schema: ModelSchema): void {
  schema.register('mddmField', {
    inheritAllFrom: '$inlineObject',
    allowAttributes: ['fieldId', 'fieldType', 'fieldLabel', 'fieldRequired', 'fieldValue'],
  });
}
