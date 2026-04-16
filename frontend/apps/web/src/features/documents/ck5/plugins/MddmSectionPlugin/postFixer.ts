import type { Editor, ModelElement, ModelWriter } from 'ckeditor5';

export function registerSectionPostFixer(editor: Editor): void {
  editor.model.document.registerPostFixer((writer: ModelWriter) => {
    let changed = false;
    const root = editor.model.document.getRoot();
    if (!root) return false;

    const walker = root.getChildren();
    for (const node of walker) {
      if (!(node as ModelElement).is('element', 'mddmSection')) continue;
      const section = node as ModelElement;
      const children = Array.from(section.getChildren()) as ModelElement[];
      const headers = children.filter((c) => c.is('element', 'mddmSectionHeader'));
      const bodies = children.filter((c) => c.is('element', 'mddmSectionBody'));

      for (const extra of headers.slice(1)) {
        writer.remove(extra);
        changed = true;
      }
      for (const extra of bodies.slice(1)) {
        writer.remove(extra);
        changed = true;
      }

      if (headers.length === 0) {
        writer.insertElement('mddmSectionHeader', section, 0);
        changed = true;
      }
      if (bodies.length === 0) {
        const body = writer.createElement('mddmSectionBody');
        writer.append(body, section);
        writer.appendElement('paragraph', body);
        changed = true;
      }
    }
    return changed;
  });
}
