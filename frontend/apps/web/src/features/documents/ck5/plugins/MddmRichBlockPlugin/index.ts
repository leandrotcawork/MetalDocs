import { Plugin, Widget } from 'ckeditor5';
import { registerRichBlockSchema } from './schema';
import { registerRichBlockConverters } from './converters';
import { InsertRichBlockCommand } from './commands/InsertRichBlockCommand';

export class MddmRichBlockPlugin extends Plugin {
  static get pluginName(): 'MddmRichBlockPlugin' {
    return 'MddmRichBlockPlugin';
  }

  static get requires(): ReadonlyArray<typeof Widget> {
    return [Widget];
  }

  init(): void {
    const { editor } = this;
    registerRichBlockSchema(editor.model.schema);
    registerRichBlockConverters(editor);
    editor.commands.add('insertMddmRichBlock', new InsertRichBlockCommand(editor));
  }
}
