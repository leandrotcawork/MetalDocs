import type { Comment } from '@metaldocs/editor-ui';
import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { toast } from 'sonner';
import { useDocumentComments } from '../useDocumentComments';

vi.mock('sonner', () => ({
  toast: {
    error: vi.fn(),
  },
}));

function makeComment(id: number): Comment {
  return {
    id,
    author: 'Alice Doe',
    initials: 'AD',
    date: '2026-04-19T10:00:00Z',
    content: [{ type: 'paragraph' }] as unknown as Comment['content'],
    done: false,
  };
}

describe('useDocumentComments add', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('keeps optimistic add on success and rolls back on failure', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ([]),
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          id: 'uuid-1',
          library_comment_id: 100,
          parent_library_id: null,
          author: 'Alice Doe',
          author_id: 'iam-abc',
          content: [{ type: 'paragraph' }],
          done: false,
          created_at: '2026-04-19T10:00:00Z',
          updated_at: '2026-04-19T10:00:00Z',
          resolved_at: null,
        }),
      } as Response)
      .mockRejectedValueOnce(new Error('network'));

    const { result } = renderHook(() => useDocumentComments('doc-1', 'Alice Doe'));
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await result.current.add(makeComment(100));
    });
    expect(result.current.comments).toHaveLength(1);

    await act(async () => {
      await result.current.add(makeComment(101));
    });
    expect(result.current.comments).toHaveLength(1);
    expect(toast.error).toHaveBeenCalled();
  });
});
