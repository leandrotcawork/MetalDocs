import { Command } from 'ckeditor5';
import type { Editor, ModelElement } from 'ckeditor5';

import { findAncestorByName } from '../../../shared/findAncestor';

export class RemoveRepeatableItemCommand extends Command {
  declare public editor: Editor;

  public override refresh(): void {
    const { model } = this.editor;
    const selection = model.document.selection;
    const position = selection.getFirstPosition();
    const item = position ? findAncestorByName(position.parent, 'mddmRepeatableItem') : null;

    if (!item) {
      this.isEnabled = false;
      return;
    }

    const repeatable = findAncestorByName(item as ModelElement, 'mddmRepeatable');
    if (!repeatable) {
      this.isEnabled = false;
      return;
    }

    const itemCount = Array.from((repeatable as ModelElement).getChildren()).length;
    const min = repeatable.getAttribute('min');
    this.isEnabled = typeof min !== 'number' || itemCount > min;
  }

  public override execute(): void {
    const { model } = this.editor;
    const selection = model.document.selection;
    const position = selection.getFirstPosition();
    const item = position ? findAncestorByName(position.parent, 'mddmRepeatableItem') : null;

    if (!item) {
      return;
    }

    model.change((writer) => {
      writer.remove(item as ModelElement);
    });
  }
}
