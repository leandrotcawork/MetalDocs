import { Command, type Editor } from 'ckeditor5';
import { uid } from '../../../shared/uid';
import type { SectionVariant } from '../../../types';

export interface InsertSectionOptions {
  variant?: SectionVariant;
  sectionId?: string;
}

export class InsertSectionCommand extends Command {
  constructor(editor: Editor) {
    super(editor);
  }

  override refresh(): void {
    const sel = this.editor.model.document.selection;
    const pos = sel.getFirstPosition();
    this.isEnabled = !!pos && this.editor.model.schema.findAllowedParent(pos, 'mddmSection') !== null;
  }

  override execute(opts: InsertSectionOptions = {}): void {
    const variant: SectionVariant = opts.variant ?? 'editable';
    const sectionId = opts.sectionId ?? uid('sec');
    const model = this.editor.model;

    model.change((writer) => {
      const section = writer.createElement('mddmSection', { sectionId, variant });
      const header = writer.createElement('mddmSectionHeader');
      const body = writer.createElement('mddmSectionBody');
      writer.append(header, section);
      writer.append(body, section);
      writer.appendElement('paragraph', body);
      // insertContent handles both inline and block contexts by auto-detecting
      // the correct insertion position (splits paragraph for block objects).
      model.insertContent(section, model.document.selection);

      if (variant === 'editable' || variant === 'mixed') {
        const range = model.createRangeIn(body);
        writer.addMarker(`restrictedEditingException:${uid('rex')}`, {
          range,
          usingOperation: true,
          affectsData: true,
        });
      }
    });
  }
}
