import { Command, type Editor } from 'ckeditor5';
import { uid } from '../../../shared/uid';

export interface InsertFieldOptions {
  fieldId?: string;
  fieldType: string;
  fieldLabel: string;
  fieldRequired?: boolean;
  fieldValue?: string;
}

export class InsertFieldCommand extends Command {
  constructor(editor: Editor) {
    super(editor);
  }

  override refresh(): void {
    const sel = this.editor.model.document.selection;
    const pos = sel.getFirstPosition();
    this.isEnabled = !!pos && this.editor.model.schema.checkChild(pos, 'mddmField');
  }

  override execute(opts: InsertFieldOptions): void {
    const { model } = this.editor;
    model.change((writer) => {
      const field = writer.createElement('mddmField', {
        fieldId: opts.fieldId ?? uid('fld'),
        fieldType: opts.fieldType,
        fieldLabel: opts.fieldLabel,
        fieldRequired: !!opts.fieldRequired,
        fieldValue: opts.fieldValue ?? '',
      });
      model.insertContent(field);
      writer.setSelection(field, 'after');
    });
  }
}
