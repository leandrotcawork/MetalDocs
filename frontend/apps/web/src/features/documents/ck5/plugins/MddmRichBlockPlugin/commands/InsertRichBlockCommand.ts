import { Command, type Editor } from 'ckeditor5';

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
      const exception = writer.createElement('restrictedEditingException');
      writer.appendElement('paragraph', exception);
      writer.append(exception, block);
      model.insertContent(block, model.document.selection);
    });
  }
}
