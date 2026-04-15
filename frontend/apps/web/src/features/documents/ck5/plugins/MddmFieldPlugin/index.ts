import { Plugin, Widget, viewToModelPositionOutsideModelElement } from 'ckeditor5';
import { registerFieldSchema } from './schema';
import { registerFieldConverters } from './converters';
import { InsertFieldCommand } from './commands/InsertFieldCommand';
import { registerInsertionButton } from '../../shared/registerInsertionButton';

export class MddmFieldPlugin extends Plugin {
  static get pluginName(): 'MddmFieldPlugin' {
    return 'MddmFieldPlugin';
  }

  static get requires(): ReadonlyArray<typeof Widget> {
    return [Widget];
  }

  init(): void {
    const editor = this.editor;

    registerFieldSchema(editor.model.schema);
    registerFieldConverters(editor);

    editor.commands.add('insertMddmField', new InsertFieldCommand(editor));

    // Inline widget position mapping: when caret moves "past" the chip in
    // the view, translate to the model position AFTER the mddmField element.
    editor.editing.mapper.on(
      'viewToModelPosition',
      viewToModelPositionOutsideModelElement(editor.model, (viewEl) =>
        viewEl.hasClass('mddm-field'),
      ),
    );

    registerInsertionButton(editor, {
      componentName: 'insertMddmField',
      commandName: 'insertMddmField',
      label: 'Insert field',
      executeOptions: { fieldType: 'text', fieldLabel: 'Field', fieldValue: '' },
    });
  }
}
