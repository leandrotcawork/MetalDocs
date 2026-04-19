import { forwardRef, useEffect, useImperativeHandle, useRef } from 'react';
import { DocxEditor, type DocxEditorRef } from '@eigenpal/docx-js-editor';
import './overrides.css';
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

    return (
      <div className="metaldocs-editor" data-mode={props.mode}>
        <DocxEditor
          ref={inner}
          documentBuffer={props.documentBuffer}
          showToolbar={props.mode !== 'readonly'}
          showRuler
          onChange={handleChange}
        />
      </div>
    );
  }
);
