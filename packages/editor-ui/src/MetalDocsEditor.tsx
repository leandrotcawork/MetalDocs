import { forwardRef, useEffect, useImperativeHandle, useRef } from 'react';
import { DocxEditor, type DocxEditorRef } from '@eigenpal/docx-js-editor';
import '@eigenpal/docx-js-editor/styles.css';
import type { MetalDocsEditorProps, MetalDocsEditorRef } from './types';

const AUTOSAVE_DEBOUNCE_MS = 1500;

export const MetalDocsEditor = forwardRef<MetalDocsEditorRef, MetalDocsEditorProps>(
  function MetalDocsEditor(props, ref) {
    const inner = useRef<DocxEditorRef>(null);
    const onAutoSaveRef = useRef(props.onAutoSave);
    const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const inFlightRef = useRef(false);

    onAutoSaveRef.current = props.onAutoSave;

    useImperativeHandle(ref, () => ({
      async getDocumentBuffer() {
        if (!inner.current) return null;
        return (await inner.current.save()) ?? null;
      },
      focus() {},
    }), []);

    useEffect(() => () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    }, []);

    const handleChange = () => {
      if (props.mode === 'readonly') return;
      if (!onAutoSaveRef.current) return;
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(async () => {
        if (inFlightRef.current) return;
        if (!inner.current) return;
        const cb = onAutoSaveRef.current;
        if (!cb) return;
        try {
          inFlightRef.current = true;
          const buf = await inner.current.save();
          if (buf) await cb(buf);
        } finally {
          inFlightRef.current = false;
        }
      }, AUTOSAVE_DEBOUNCE_MS);
    };

    const libMode = props.mode === 'readonly' ? 'viewing' : 'editing';

    return (
      <DocxEditor
        ref={inner}
        documentBuffer={props.documentBuffer}
        mode={libMode}
        author={props.author ?? props.userId}
        documentName={props.documentName}
        documentNameEditable={props.documentNameEditable ?? (libMode === 'editing')}
        onDocumentNameChange={props.onDocumentNameChange}
        comments={props.comments}
        onCommentsChange={props.onCommentsChange}
        onCommentAdd={props.onCommentAdd}
        onCommentResolve={props.onCommentResolve}
        onCommentDelete={props.onCommentDelete}
        onCommentReply={props.onCommentReply}
        renderTitleBarRight={props.renderTitleBarRight}
        showRuler
        showMarginGuides
        showOutlineButton
        showPrintButton
        showZoomControl
        onChange={handleChange}
      />
    );
  }
);
