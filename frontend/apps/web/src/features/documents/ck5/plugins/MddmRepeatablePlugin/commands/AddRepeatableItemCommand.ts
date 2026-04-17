import { Command } from 'ckeditor5';
import type { Editor, ModelElement } from 'ckeditor5';

import { findAncestorByName } from '../../../shared/findAncestor';

export class AddRepeatableItemCommand extends Command {
  declare public editor: Editor;

  public override refresh(): void {
    const { model } = this.editor;
    const selection = model.document.selection;
    const position = selection.getFirstPosition();
    const repeatable = position ? findAncestorByName(position.parent, 'mddmRepeatable') : null;

    if (!repeatable) {
      this.isEnabled = false;
      return;
    }

    const itemCount = Array.from(repeatable.getChildren()).length;
    const max = repeatable.getAttribute('max');
    this.isEnabled = max === Infinity || typeof max !== 'number' || itemCount < max;
  }

  public override execute(): void {
    const { model } = this.editor;
    const selection = model.document.selection;
    const position = selection.getFirstPosition();
    const repeatable = position ? findAncestorByName(position.parent, 'mddmRepeatable') : null;

    if (!repeatable) {
      return;
    }

    model.change((writer) => {
      const item = writer.createElement('mddmRepeatableItem');
      const exception = writer.createElement('restrictedEditingException');
      writer.appendElement('paragraph', exception);
      writer.append(exception, item);
      writer.append(item, repeatable as ModelElement);
    });
  }
}
