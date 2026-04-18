import { forwardRef, useImperativeHandle, useRef } from 'react';
import { DocxEditor, type DocxEditorRef } from '@eigenpal/docx-js-editor';
import './overrides.css';
import type { MetalDocsEditorProps, MetalDocsEditorRef } from './types';

export const MetalDocsEditor = forwardRef<MetalDocsEditorRef, MetalDocsEditorProps>(
  function MetalDocsEditor(props, ref) {
    const inner = useRef<DocxEditorRef>(null);

    useImperativeHandle(ref, () => ({
      async getDocumentBuffer() {
        if (!inner.current) return null;
        return (await inner.current.save()) ?? null;
      },
      focus() {},
    }), []);

    return (
      <div className="metaldocs-editor" data-mode={props.mode}>
        <DocxEditor
          ref={inner}
          documentBuffer={props.documentBuffer}
          showToolbar={props.mode !== 'readonly'}
          showRuler
        />
      </div>
    );
  }
);
