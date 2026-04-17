import { v4 as uuidv4 } from 'uuid';
import type { Editor, Element } from 'ckeditor5';
import { PAGINABLE_ELEMENT_NAMES } from './schema';

const PAGINABLE = new Set<string>(PAGINABLE_ELEMENT_NAMES);

/**
 * Post-fixer contract:
 * 1. Any paginable element without `mddmBid` → mint fresh UUID.
 * 2. If multiple paginable elements share a bid in a single document pass
 *    (i.e. split produced clones), keep the FIRST occurrence and re-mint
 *    subsequent ones. This preserves the survivor-keeps-bid rule.
 *
 * Note: merge is handled naturally by CK5's merge operation — the absorbed
 * element is removed, so its bid disappears with it.
 */
export function registerBidPostFixer(editor: Editor): void {
  editor.model.document.registerPostFixer(writer => {
    const root = editor.model.document.getRoot();
    if (!root) return false;

    let changed = false;
    const seen = new Map<string, Element>();

    for (const { item } of editor.model.createRangeIn(root)) {
      if (!item.is('element')) continue;
      const el = item as Element;
      if (!PAGINABLE.has(el.name)) continue;

      const bid = el.getAttribute('mddmBid') as string | undefined;
      if (!bid) {
        writer.setAttribute('mddmBid', uuidv4(), el);
        changed = true;
        continue;
      }
      if (seen.has(bid)) {
        // Duplicate: re-mint on the later element (survivor rule).
        writer.setAttribute('mddmBid', uuidv4(), el);
        changed = true;
        continue;
      }
      seen.set(bid, el);
    }
    return changed;
  });
}
