import { Plugin, Widget } from 'ckeditor5';
import { registerSectionSchema } from './schema';
import { registerSectionPostFixer } from './postFixer';
import { registerSectionConverters } from './converters';
import { InsertSectionCommand } from './commands/InsertSectionCommand';

export class MddmSectionPlugin extends Plugin {
  static get pluginName(): 'MddmSectionPlugin' {
    return 'MddmSectionPlugin';
  }

  static get requires(): ReadonlyArray<typeof Widget> {
    return [Widget];
  }

  init(): void {
    const editor = this.editor;
    registerSectionSchema(editor.model.schema);
    registerSectionPostFixer(editor);
    registerSectionConverters(editor);
    editor.commands.add('insertMddmSection', new InsertSectionCommand(editor));
  }
}
