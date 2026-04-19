import type { Comment } from '@metaldocs/editor-ui';
import { describe, expect, it } from 'vitest';
import { partitionByMarkers } from '../useDocumentComments';

describe('partitionByMarkers', () => {
  it('splits live and orphan comments by marker ids', () => {
    const comments: Comment[] = [
      { id: 1, author: 'A', content: [] as unknown as Comment['content'] },
      { id: 2, author: 'B', content: [] as unknown as Comment['content'] },
      { id: 3, author: 'C', content: [] as unknown as Comment['content'] },
    ];

    const result = partitionByMarkers(comments, new Set([1, 3]));

    expect(result.live.map((c) => c.id)).toEqual([1, 3]);
    expect(result.orphans.map((c) => c.id)).toEqual([2]);
  });
});
