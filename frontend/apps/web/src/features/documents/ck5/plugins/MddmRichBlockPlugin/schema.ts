import type { ModelSchema } from 'ckeditor5';

export function registerRichBlockSchema(schema: ModelSchema): void {
  schema.register('mddmRichBlock', {
    inheritAllFrom: '$container',
    allowIn: ['$root'],
  });
}
