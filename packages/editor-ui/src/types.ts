import type { ReactNode } from 'react';
import type { Comment } from '@eigenpal/docx-js-editor';

export type EditorMode = 'template-draft' | 'document-edit' | 'readonly';

export interface MetalDocsEditorProps {
  documentId?: string;
  documentBuffer?: ArrayBuffer;
  mode: EditorMode;
  userId: string;
  author?: string;
  documentName?: string;
  documentNameEditable?: boolean;
  onDocumentNameChange?: (name: string) => void;
  comments?: Comment[];
  onCommentsChange?: (comments: Comment[]) => void;
  onCommentAdd?: (c: Comment) => void;
  onCommentResolve?: (c: Comment) => void;
  onCommentDelete?: (c: Comment) => void;
  onCommentReply?: (reply: Comment, parent: Comment) => void;
  renderTitleBarRight?: () => ReactNode;
  onAutoSave?: (buf: ArrayBuffer) => Promise<void>;
  onLockLost?: () => void;
}

export interface MetalDocsEditorRef {
  getDocumentBuffer(): Promise<ArrayBuffer | null>;
  focus(): void;
}
