import { describe, it, expect } from 'vitest';
import { findAncestorByName } from '../findAncestor';

type FakeNode = {
  name?: string;
  parent?: FakeNode | null;
  is: (kind: 'element', name?: string) => boolean;
};

function node(name: string, parent: FakeNode | null): FakeNode {
  const n: FakeNode = {
    name,
    parent,
    is(kind, checkName) {
      return kind === 'element' && (!checkName || checkName === name);
    },
  };
  return n;
}

describe('findAncestorByName', () => {
  it('returns node itself if it matches', () => {
    const target = node('mddmSection', null);
    expect(findAncestorByName(target as never, 'mddmSection')).toBe(target);
  });

  it('walks up until a match', () => {
    const root = node('root', null);
    const section = node('mddmSection', root);
    const body = node('mddmSectionBody', section);
    const para = node('paragraph', body);
    expect(findAncestorByName(para as never, 'mddmSection')).toBe(section);
  });

  it('returns null if no match found', () => {
    const root = node('root', null);
    const para = node('paragraph', root);
    expect(findAncestorByName(para as never, 'mddmSection')).toBeNull();
  });
});
