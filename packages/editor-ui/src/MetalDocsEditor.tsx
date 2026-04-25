import { forwardRef, useEffect, useImperativeHandle, useMemo, useRef } from 'react';
import { DocxEditor, PluginHost, templatePlugin, type DocxEditorRef, type ReactEditorPlugin } from '@eigenpal/docx-js-editor';
import { createOutlinePlugin } from './plugins/OutlinePlugin';
import '@eigenpal/docx-js-editor/styles.css';
import type { MetalDocsEditorProps, MetalDocsEditorRef } from './types';
import { buildSidebarModelPlugin } from './plugins/sidebarModelBridge';

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

    const outlinePlugin = useMemo(() => createOutlinePlugin(), []);

    const libMode = props.mode === 'readonly' ? 'viewing' : 'editing';
    const plugins: ReactEditorPlugin[] = [
      templatePlugin,
      ...(props.mode !== 'readonly' ? [outlinePlugin] : []),
      ...(props.sidebarModel ? [buildSidebarModelPlugin(props.sidebarModel)] : []),
      ...(props.externalPlugins ?? []),
    ];

    return (
      <PluginHost plugins={plugins}>
        <DocxEditor
          ref={inner}
          documentBuffer={props.documentBuffer}
          mode={libMode}
          author={props.author}
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
      </PluginHost>
    );
  }
);
