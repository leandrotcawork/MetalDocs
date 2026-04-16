import { Command } from 'ckeditor5';
import type { Editor } from 'ckeditor5';

import { uid } from '../../../shared/uid';

type InsertRepeatableOptions = {
  repeatableId?: string;
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

  public override execute(options: InsertRepeatableOptions = {}): void {
    const {
      repeatableId = uid('rep'),
      label = '',
      min: rawMin = 0,
      max: rawMax = Infinity,
      numberingStyle = 'decimal',
      initialCount,
    } = options;
    const min = Math.max(0, rawMin);
    const max = rawMax > 0 ? rawMax : Infinity;
    const count = Math.max(min, Math.min(initialCount ?? Math.max(min, 1), isFinite(max) ? max : Infinity));
    const { model } = this.editor;

    model.change((writer) => {
      const repeatable = writer.createElement('mddmRepeatable', {
        repeatableId,
        label,
        min,
        max,
        numberingStyle,
      });

      for (let i = 0; i < count; i += 1) {
        const item = writer.createElement('mddmRepeatableItem');
        const exception = writer.createElement('restrictedEditingException');
        writer.appendElement('paragraph', exception);
        writer.append(exception, item);
        writer.append(item, repeatable);
      }

      model.insertObject(repeatable, null, null, { setSelection: 'on' });
    });
  }
}
