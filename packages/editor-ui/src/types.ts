export type EditorMode = 'template-draft' | 'document-edit' | 'readonly';

export interface MetalDocsEditorProps {
  documentId?: string;
  documentBuffer?: ArrayBuffer;
  mode: EditorMode;
  schema?: unknown;
  onAutoSave?: (buf: ArrayBuffer) => Promise<void>;
  onLockLost?: () => void;
  userId: string;
}

export interface MetalDocsEditorRef {
  getDocumentBuffer(): Promise<ArrayBuffer | null>;
  focus(): void;
}
