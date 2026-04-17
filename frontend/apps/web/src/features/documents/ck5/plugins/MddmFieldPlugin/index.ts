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

    // Post-fixer: strip GHS htmlSpan attribute injected onto mddmField elements.
    // GHS adds htmlSpan to any model element upcast from <span>, which causes GHS
    // dataDowncast to emit an outer wrapper <span data-field-*> around our
    // MddmFieldPlugin output, producing double-span in getData().
    const GHS_SPAN_ATTRS = ['htmlSpan', 'htmlSpanAttributes'] as const;

    editor.model.document.registerPostFixer((writer) => {
      let changed = false;
      const root = editor.model.document.getRoot();
      if (!root) return false;

      for (const { item } of editor.model.createRangeIn(root)) {
        if (!item.is('element', 'mddmField')) continue;
        for (const attr of GHS_SPAN_ATTRS) {
          if (item.hasAttribute(attr)) {
            writer.removeAttribute(attr, item);
            changed = true;
          }
        }
      }
      return changed;
    });

    registerInsertionButton(editor, {
      componentName: 'insertMddmField',
      commandName: 'insertMddmField',
      label: 'Insert field',
      executeOptions: { fieldType: 'text', fieldLabel: 'Field', fieldValue: '' },
    });
  }
}
