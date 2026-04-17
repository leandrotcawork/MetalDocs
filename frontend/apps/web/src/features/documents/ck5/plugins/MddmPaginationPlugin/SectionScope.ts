import type { Element, Position as ModelPosition } from 'ckeditor5';

/**
 * Walks the parent chain from `position.parent` upward and returns the
 * nearest `mddmSection` ancestor, or null if none is found.
 */
export function findEnclosingSection(position: ModelPosition): Element | null {
  let node: unknown = position.parent;
  while (node) {
    const n = node as { is?: (t: string) => boolean; name?: string; parent?: unknown };
    if (typeof n.is === 'function' && n.is('element') && n.name === 'mddmSection') {
      return n as unknown as Element;
    }
    node = n.parent;
  }
  return null;
}
