import type { Schema } from 'ckeditor5';

export function registerRichBlockSchema(schema: Schema): void {
  schema.register('mddmRichBlock', {
    inheritAllFrom: '$container',
    allowIn: ['$root'],
  });
}
