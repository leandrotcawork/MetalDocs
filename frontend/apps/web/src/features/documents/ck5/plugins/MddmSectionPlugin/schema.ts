import type { ModelSchema } from 'ckeditor5';

export function registerSectionSchema(schema: ModelSchema): void {
  schema.register('mddmSection', {
    inheritAllFrom: '$blockObject',
    allowChildren: ['mddmSectionHeader', 'mddmSectionBody'],
    allowAttributes: ['sectionId', 'variant'],
  });

  schema.register('mddmSectionHeader', {
    isLimit: true,
    allowIn: 'mddmSection',
    allowContentOf: '$block',
  });

  schema.register('mddmSectionBody', {
    isLimit: true,
    allowIn: 'mddmSection',
    allowContentOf: '$root',
  });
}
