import { Command } from 'ckeditor5';
import type { Editor, ModelElement } from 'ckeditor5';

import { uid } from '../../../shared/uid';

type InsertRepeatableOptions = {
  repeatableId: string;
  label?: string;
  min?: number;
  max?: number;
  numberingStyle?: string;
  initialCount?: number;
};

export class InsertRepeatableCommand extends Command {
  declare public editor: Editor;

  public override refresh(): void {
    const { model } = this.editor;
    const selection = model.document.selection;
    const allowedParent = model.schema.findAllowedParent(selection.getFirstPosition()!, 'mddmRepeatable');

    this.isEnabled = !!allowedParent;
  }

  public override execute(options: InsertRepeatableOptions): void {
    const {
      repeatableId,
      label = '',
      min = 1,
      max = Infinity,
      numberingStyle = '',
      initialCount = 1,
    } = options;
    const { model } = this.editor;

    model.change((writer) => {
      const repeatable = writer.createElement('mddmRepeatable', {
        repeatableId,
        label,
        min,
        max,
        numberingStyle,
      });

      for (let i = 0; i < initialCount; i += 1) {
        const item = writer.createElement('mddmRepeatableItem');
        const paragraph = writer.createElement('paragraph');
        writer.append(paragraph, item);
        writer.append(item, repeatable);
      }

      model.insertObject(repeatable, null, null, { setSelection: 'on' });

      for (const item of repeatable.getChildren()) {
        writer.addMarker(`restrictedEditingException:${uid()}`, {
          range: writer.createRangeOn(item as ModelElement),
          usingOperation: false,
          affectsData: true,
        });
      }
    });
  }
}
