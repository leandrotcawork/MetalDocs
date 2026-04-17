import { v4 as uuidv4 } from 'uuid';
import type { Editor, DocumentFragment, Element } from 'ckeditor5';
import { PAGINABLE_ELEMENT_NAMES } from './schema';

const PAGINABLE = new Set<string>(PAGINABLE_ELEMENT_NAMES);

/**
 * After clipboard upcast produces a model DocumentFragment, walk it and
 * re-mint any bid that collides with a bid already present in the document.
 */
export function registerBidClipboardHandler(editor: Editor): void {
  editor.plugins.get('ClipboardPipeline').on('contentInsertion', (_evt: unknown, data: { content: DocumentFragment }) => {
    const fragment = data.content;
    const existing = collectDocumentBids(editor);

    editor.model.change(writer => {
      for (const { item } of editor.model.createRangeIn(fragment)) {
        if (!item.is('element')) continue;
        const el = item as Element;
        if (!PAGINABLE.has(el.name)) continue;
        const bid = el.getAttribute('mddmBid') as string | undefined;
        if (bid && existing.has(bid)) {
          writer.setAttribute('mddmBid', uuidv4(), el);
        }
      }
    });
  });
}

function collectDocumentBids(editor: Editor): Set<string> {
  const out = new Set<string>();
  const root = editor.model.document.getRoot();
  if (!root) return out;
  for (const { item } of editor.model.createRangeIn(root)) {
    if (!item.is('element')) continue;
    const bid = (item as Element).getAttribute('mddmBid') as string | undefined;
    if (bid) out.add(bid);
  }
  return out;
}
