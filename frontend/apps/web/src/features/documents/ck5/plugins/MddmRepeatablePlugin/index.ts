import { Plugin, Widget } from 'ckeditor5';

import { registerRepeatableConverters } from './converters';
import { AddRepeatableItemCommand } from './commands/AddRepeatableItemCommand';
import { InsertRepeatableCommand } from './commands/InsertRepeatableCommand';
import { RemoveRepeatableItemCommand } from './commands/RemoveRepeatableItemCommand';
import { registerRepeatableSchema } from './schema';
import { registerInsertionButton } from '../../shared/registerInsertionButton';

export class MddmRepeatablePlugin extends Plugin {
  public static get pluginName(): 'MddmRepeatablePlugin' {
    return 'MddmRepeatablePlugin';
  }

  public static get requires() {
    return [Widget] as const;
  }

  public init(): void {
    const { editor } = this;

    registerRepeatableSchema(editor.model.schema);
    registerRepeatableConverters(editor);

    editor.commands.add('insertMddmRepeatable', new InsertRepeatableCommand(editor));
    editor.commands.add('addMddmRepeatableItem', new AddRepeatableItemCommand(editor));
    editor.commands.add('removeMddmRepeatableItem', new RemoveRepeatableItemCommand(editor));

    registerInsertionButton(editor, {
      componentName: 'insertMddmRepeatable',
      commandName: 'insertMddmRepeatable',
      label: 'Insert repeatable',
      executeOptions: { min: 1, max: 10, initialCount: 1 },
    });

    const LIST_ATTRS = ['listItemId', 'listIndent', 'listType', 'htmlLiAttributes'] as const;

    editor.model.document.registerPostFixer((writer) => {
      let changed = false;
      const root = editor.model.document.getRoot();
      if (!root) return false;

      for (const { item } of editor.model.createRangeIn(root)) {
        if (!item.is('element', 'mddmRepeatableItem')) continue;
        for (const attr of LIST_ATTRS) {
          if (item.hasAttribute(attr)) {
            writer.removeAttribute(attr, item);
            changed = true;
          }
        }
      }
      return changed;
    });
  }
}
