import type { Editor, Element, Position as ModelPosition } from 'ckeditor5';
import { PAGINABLE_ELEMENT_NAMES } from '../MddmBlockIdentityPlugin/schema';
import type { BreakCandidate } from './types';

const PAGINABLE = new Set<string>(PAGINABLE_ELEMENT_NAMES);
const KEEP_WITH_NEXT = new Set(['heading1', 'heading2', 'heading3']);

/**
 * Walks the model from `from` to the document end and emits one BreakCandidate
 * after each paginable block.
 *
 * Rules:
 * - heading1/2/3 → keep-with-next: skipped (no candidate emitted)
 * - blocks without a bid → skipped
 * - non-paginable elements → skipped
 */
export function planBreaks(
  editor: Editor,
  from: ModelPosition,
  walkRoot?: Element,
): BreakCandidate[] {
  const root = editor.model.document.getRoot();
  if (!root) return [];

  const container: Element | null = walkRoot ?? (root as unknown as Element);
  if (!container) return [];

  const out: BreakCandidate[] = [];
  const startIdx = walkRoot ? 0 : (from.path[0] ?? 0);

  for (let i = startIdx; i < container.childCount; i++) {
    const node = container.getChild(i);
    if (!node || !node.is('element')) continue;
    const el = node as Element;
    if (!PAGINABLE.has(el.name)) continue;
    if (KEEP_WITH_NEXT.has(el.name)) continue;
    const bid = el.getAttribute('mddmBid') as string | undefined;
    if (!bid) continue;

    const afterPos = editor.model.createPositionAfter(el);
    out.push({
      afterBid: bid,
      modelPath: Array.from(afterPos.path),
      keepWithNext: false,
    });
  }
  return out;
}
