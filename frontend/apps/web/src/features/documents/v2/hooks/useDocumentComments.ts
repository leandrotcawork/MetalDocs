import { useCallback, useEffect, useMemo, useState, type Dispatch, type SetStateAction } from 'react';
import type { Comment } from '@metaldocs/editor-ui';
import { toast } from 'sonner';
import { createComment, deleteComment, listComments, patchComment, type CommentRow } from '../api/documentsV2';

function toInitials(author: string): string {
  return author
    .trim()
    .split(/\s+/)
    .filter(Boolean)
    .map((token) => token[0]?.toUpperCase() ?? '')
    .join('')
    .slice(0, 2);
}

function rowToLibrary(row: CommentRow): Comment {
  return {
    id: row.library_comment_id,
    parentId: row.parent_library_id ?? undefined,
    author: row.author,
    initials: toInitials(row.author),
    date: row.created_at,
    content: row.content as Comment['content'],
    done: row.done,
  };
}

function rowsToLibrary(rows: CommentRow[]): Comment[] {
  return rows.map(rowToLibrary);
}

function toPayloadContent(comment: Comment): unknown[] {
  return comment.content as unknown[];
}

export function partitionByMarkers(comments: Comment[], markerIds: Set<number>): { live: Comment[]; orphans: Comment[] } {
  const live: Comment[] = [];
  const orphans: Comment[] = [];
  for (const comment of comments) {
    if (markerIds.has(comment.id)) live.push(comment);
    else orphans.push(comment);
  }
  return { live, orphans };
}

export function useDocumentComments(documentID: string, authorDisplay: string): {
  comments: Comment[];
  orphans: Comment[];
  loading: boolean;
  add: (c: Comment) => Promise<void>;
  resolve: (c: Comment) => Promise<void>;
  reopen: (c: Comment) => Promise<void>;
  remove: (c: Comment) => Promise<void>;
  reply: (replyC: Comment, parent: Comment) => Promise<void>;
  setComments: Dispatch<SetStateAction<Comment[]>>;
} {
  const [comments, setComments] = useState<Comment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const orphans = useMemo(() => [] as Comment[], []);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      try {
        const rows = await listComments(documentID);
        if (!cancelled) setComments(rowsToLibrary(rows));
      } catch {
        if (!cancelled) {
          setError('Failed to load comments.');
          toast.error('Failed to load comments.');
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [documentID]);

  const add = useCallback(async (c: Comment) => {
    const snapshot = comments;
    setComments((prev) => [...prev, c]);
    try {
      const row = await createComment(documentID, {
        library_comment_id: c.id,
        author_display: authorDisplay,
        content: toPayloadContent(c),
      });
      const serverComment = rowToLibrary(row);
      setComments((prev) => prev.map((item) => (item.id === c.id ? serverComment : item)));
    } catch {
      setComments(snapshot);
      toast.error('Failed to add comment.');
    }
  }, [authorDisplay, comments, documentID]);

  const resolve = useCallback(async (c: Comment) => {
    const snapshot = comments;
    setComments((prev) => prev.map((item) => (item.id === c.id ? { ...item, done: true } : item)));
    try {
      const row = await patchComment(documentID, c.id, { done: true });
      const serverComment = rowToLibrary(row);
      setComments((prev) => prev.map((item) => (item.id === c.id ? serverComment : item)));
    } catch {
      setComments(snapshot);
      toast.error('Failed to resolve comment.');
    }
  }, [comments, documentID]);

  const reopen = useCallback(async (c: Comment) => {
    const snapshot = comments;
    setComments((prev) => prev.map((item) => (item.id === c.id ? { ...item, done: false } : item)));
    try {
      const row = await patchComment(documentID, c.id, { done: false });
      const serverComment = rowToLibrary(row);
      setComments((prev) => prev.map((item) => (item.id === c.id ? serverComment : item)));
    } catch {
      setComments(snapshot);
      toast.error('Failed to reopen comment.');
    }
  }, [comments, documentID]);

  const remove = useCallback(async (c: Comment) => {
    const snapshot = comments;
    setComments((prev) => prev.filter((item) => item.id !== c.id));
    try {
      await deleteComment(documentID, c.id);
    } catch {
      setComments(snapshot);
      toast.error('Failed to delete comment.');
    }
  }, [comments, documentID]);

  const reply = useCallback(async (replyC: Comment, parent: Comment) => {
    const snapshot = comments;
    setComments((prev) => [...prev, { ...replyC, parentId: parent.id }]);
    try {
      const row = await createComment(documentID, {
        library_comment_id: replyC.id,
        parent_library_id: parent.id,
        author_display: authorDisplay,
        content: toPayloadContent(replyC),
      });
      const serverComment = rowToLibrary(row);
      setComments((prev) => prev.map((item) => (item.id === replyC.id ? serverComment : item)));
    } catch {
      setComments(snapshot);
      toast.error('Failed to reply to comment.');
    }
  }, [authorDisplay, comments, documentID]);

  return { comments, orphans, loading, add, resolve, reopen, remove, reply, setComments };
}
