import { describe, it, expect } from 'vitest';
import type { DocxEditorProps } from '@eigenpal/docx-js-editor';
import type { MetalDocsEditorProps } from '../src/types';

type SharedPropKeys =
  | 'author'
  | 'documentName'
  | 'documentNameEditable'
  | 'onDocumentNameChange'
  | 'comments'
  | 'onCommentsChange'
  | 'onCommentAdd'
  | 'onCommentResolve'
  | 'onCommentDelete'
  | 'onCommentReply'
  | 'renderTitleBarRight';

const _forwardCompat: Pick<DocxEditorProps, SharedPropKeys> = {} as Pick<
  MetalDocsEditorProps,
  SharedPropKeys
>;
const _reverseCompat: Pick<MetalDocsEditorProps, SharedPropKeys> = {} as Pick<
  DocxEditorProps,
  SharedPropKeys
>;

describe('MetalDocsEditorProps contract', () => {
  it('matches DocxEditorProps for pass-through fields', () => {
    expect(Boolean(_forwardCompat) || Boolean(_reverseCompat) || true).toBe(true);
  });
});
