import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useDocumentComments } from '../useDocumentComments';

describe('useDocumentComments load', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('loads rows and maps to library comments', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => ([
        {
          id: 'uuid-root',
          library_comment_id: 42,
          parent_library_id: null,
          author: 'Alice Doe',
          author_id: 'iam-abc',
          content: [{ type: 'paragraph' }],
          done: false,
          created_at: '2026-04-19T10:00:00Z',
          updated_at: '2026-04-19T10:00:00Z',
          resolved_at: null,
        },
        {
          id: 'uuid-reply',
          library_comment_id: 43,
          parent_library_id: 42,
          author: 'Bob Smith',
          author_id: 'iam-def',
          content: [{ type: 'paragraph' }],
          done: true,
          created_at: '2026-04-19T10:01:00Z',
          updated_at: '2026-04-19T10:01:00Z',
          resolved_at: '2026-04-19T10:02:00Z',
        },
      ]),
    } as Response);

    const { result } = renderHook(() => useDocumentComments('doc-1', 'Alice Doe'));

    await waitFor(() => expect(result.current.comments.length).toBe(2));

    expect(fetchSpy).toHaveBeenCalledWith('/api/v2/documents/doc-1/comments');
    expect(typeof result.current.comments[0].id).toBe('number');
    expect(result.current.comments[1].parentId).toBe(42);
    expect(typeof result.current.comments[1].done).toBe('boolean');
  });
});
