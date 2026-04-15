import { Command, type Editor } from 'ckeditor5';
import { uid } from '../../../shared/uid';

export class InsertRichBlockCommand extends Command {
  constructor(editor: Editor) {
    super(editor);
  }

  override refresh(): void {
    const pos = this.editor.model.document.selection.getFirstPosition();
    this.isEnabled =
      !!pos && this.editor.model.schema.findAllowedParent(pos, 'mddmRichBlock') !== null;
  }

  override execute(): void {
    const { model } = this.editor;
    model.change((writer) => {
      const block = writer.createElement('mddmRichBlock');
      writer.appendElement('paragraph', block);
      model.insertContent(block, model.document.selection);
      const range = model.createRangeIn(block);
      writer.addMarker(`restrictedEditingException:${uid('rb')}`, {
        range,
        usingOperation: true,
        affectsData: true,
      });
    });
  }
}
