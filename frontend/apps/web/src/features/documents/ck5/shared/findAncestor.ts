export interface NodeLike {
  parent: NodeLike | null;
  is(kind: 'element', name?: string): boolean;
}

export function findAncestorByName<T extends NodeLike>(
  start: T | null,
  name: string,
): T | null {
  let node: NodeLike | null = start;
  while (node) {
    if (node.is('element', name)) {
      return node as T;
    }
    node = node.parent;
  }
  return null;
}
