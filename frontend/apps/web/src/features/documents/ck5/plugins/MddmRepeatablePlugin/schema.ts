import type { Schema } from 'ckeditor5';

export function registerRepeatableSchema(schema: Schema): void {
  schema.register('mddmRepeatable', {
    inheritAllFrom: '$blockObject',
    allowChildren: ['mddmRepeatableItem'],
    allowAttributes: ['repeatableId', 'label', 'min', 'max', 'numberingStyle'],
  });

  schema.register('mddmRepeatableItem', {
    inheritAllFrom: '$container',
    allowIn: 'mddmRepeatable',
    isLimit: true,
  });

  schema.addChildCheck((context, def) => {
    if (context.endsWith('mddmRepeatableItem') && def.name === 'mddmRepeatable') {
      return false;
    }

    return undefined;
  });
}
