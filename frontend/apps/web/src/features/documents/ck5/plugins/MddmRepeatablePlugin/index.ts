import { Plugin, Widget } from 'ckeditor5';

import { registerRepeatableConverters } from './converters';
import { AddRepeatableItemCommand } from './commands/AddRepeatableItemCommand';
import { InsertRepeatableCommand } from './commands/InsertRepeatableCommand';
import { RemoveRepeatableItemCommand } from './commands/RemoveRepeatableItemCommand';
import { registerRepeatableSchema } from './schema';

export class MddmRepeatablePlugin extends Plugin {
  public static get pluginName(): 'MddmRepeatablePlugin' {
    return 'MddmRepeatablePlugin';
  }

  public static get requires() {
    return [Widget] as const;
  }

  public override init(): void {
    const { editor } = this;

    registerRepeatableSchema(editor.model.schema);
    registerRepeatableConverters(editor);

    editor.commands.add('insertRepeatable', new InsertRepeatableCommand(editor));
    editor.commands.add('addRepeatableItem', new AddRepeatableItemCommand(editor));
    editor.commands.add('removeRepeatableItem', new RemoveRepeatableItemCommand(editor));
  }
}
