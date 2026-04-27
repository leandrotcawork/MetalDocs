import type { ReactNode } from 'react';
import type { Comment, ReactEditorPlugin } from '@eigenpal/docx-js-editor';
import type { SidebarModel } from './plugins/mergefieldPlugin';

export type EditorMode = 'template-draft' | 'document-edit' | 'readonly';

export interface MetalDocsEditorProps {
  documentId?: string;
  documentBuffer?: ArrayBuffer;
  mode: EditorMode;
  author: string;
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
  sidebarModel?: SidebarModel;
  externalPlugins?: ReactEditorPlugin[];
  onAutoSave?: (buf: ArrayBuffer) => Promise<void>;
  onLockLost?: () => void;
  showRuler?: boolean;
}

export interface MetalDocsEditorRef {
  getDocumentBuffer(): Promise<ArrayBuffer | null>;
  focus(): void;
}
