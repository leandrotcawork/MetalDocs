import { type PartialBlock } from "@blocknote/core";
import { BlockNoteView } from "@blocknote/mantine";
import {
  BlockNoteViewEditor,
  FilePanelController,
  FormattingToolbar,
  getFormattingToolbarItems,
  useCreateBlockNote,
  type FilePanelProps,
} from "@blocknote/react";
import { useEffect, useMemo, type CSSProperties } from "react";
import "@blocknote/core/fonts/inter.css";
import "@blocknote/mantine/style.css";
import "./mddm-editor-global.css";
import { getAttachmentDownloadURL, uploadAttachment } from "../../../api/documents";
import { mddmSchema } from "./schema";
import styles from "./MDDMEditor.module.css";
import { defaultLayoutTokens, tokensToCssVars } from "./engine/layout-ir";
import { setEditorTokens } from "./engine/editor-tokens";
import { MddmUppyFilePanel } from "./MddmUppyFilePanel";

export type MDDMTheme = {
  accent?: string;
  accentLight?: string;
  accentDark?: string;
  accentBorder?: string;
};

export type MDDMEditorProps = {
  initialContent?: PartialBlock[];
  onChange?: (blocks: unknown[]) => void;
  readOnly?: boolean;
  theme?: MDDMTheme;
  onEditorReady?: (editor: unknown) => void;
  documentId?: string;
};

export function MDDMEditor({
  initialContent,
  onChange,
  readOnly,
  theme,
  onEditorReady,
  documentId,
}: MDDMEditorProps) {
  const uploadFile = useMemo(() => {
    if (!documentId) return undefined;

    return async (file: File): Promise<string> => {
      const attachment = await uploadAttachment(documentId, file);
      return `/api/v1/documents/${documentId}/attachments/${attachment.attachmentId}/download-url`;
    };
  }, [documentId]);

  const resolveFileUrl = useMemo(() => {
    if (!documentId) return undefined;

    return async (url: string): Promise<string> => {
      const match = url.match(
        /^\/api\/v1\/documents\/([^/]+)\/attachments\/([^/]+)\/download-url$/,
      );
      if (!match) {
        return url;
      }

      const [, urlDocumentId, attachmentId] = match;
      if (urlDocumentId !== documentId) {
        return url;
      }

      try {
        const response = await getAttachmentDownloadURL(documentId, attachmentId);
        return response.downloadUrl || url;
      } catch {
        return url;
      }
    };
  }, [documentId]);

  const editor = useCreateBlockNote({
    schema: mddmSchema,
    initialContent: initialContent?.length ? initialContent : undefined,
    tables: {
      headers: true,
      cellBackgroundColor: true,
    },
    uploadFile,
    resolveFileUrl,
  });

  useEffect(() => {
    if (import.meta.env.DEV) {
      (window as any).__mddmEditor = editor;
      return () => { delete (window as any).__mddmEditor; };
    }
    return undefined;
  }, [editor]);

  const tokens = useMemo(() => {
    if (!theme) return defaultLayoutTokens;
    return {
      ...defaultLayoutTokens,
      theme: {
        ...defaultLayoutTokens.theme,
        ...(theme.accent ? { accent: theme.accent } : {}),
        ...(theme.accentLight ? { accentLight: theme.accentLight } : {}),
        ...(theme.accentDark ? { accentDark: theme.accentDark } : {}),
        ...(theme.accentBorder ? { accentBorder: theme.accentBorder } : {}),
      },
    };
  }, [theme]);

  const cssVars = useMemo(() => tokensToCssVars(tokens), [tokens]);

  useEffect(() => {
    setEditorTokens(editor, tokens);
  }, [editor, tokens]);

  useEffect(() => {
    const root = (editor as any)?._tiptapEditor?.view?.dom;
    if (!(root instanceof HTMLElement)) {
      return undefined;
    }

    const lockHeaders = () => {
      root.querySelectorAll("th").forEach((headerCell) => {
        (headerCell as HTMLElement).contentEditable = "false";
      });
    };

    lockHeaders();

    const observer = new MutationObserver(() => {
      lockHeaders();
    });
    observer.observe(root, { childList: true, subtree: true });

    return () => {
      observer.disconnect();
    };
  }, [editor]);

  useEffect(() => {
    onEditorReady?.(editor);
  }, [editor, onEditorReady]);

  // Uppy file panel — only when a documentId is available (upload endpoint requires it)
  const uppyFilePanel = useMemo(() => {
    if (!documentId) return undefined;
    return (props: FilePanelProps) => (
      <MddmUppyFilePanel {...props} documentId={documentId} />
    );
  }, [documentId]);

  // Place cursor in first inline-editable block on mount so toolbar items
  // have a ProseMirror selection and render immediately.
  useEffect(() => {
    if (readOnly) return;

    function findFirstInlineBlock(
      blocks: (typeof editor.document),
    ): (typeof editor.document)[number] | undefined {
      for (const block of blocks) {
        if (Array.isArray(block.content)) {
          return block;
        }
        if (block.children.length > 0) {
          const found = findFirstInlineBlock(block.children);
          if (found) return found;
        }
      }
      return undefined;
    }

    const firstBlock = findFirstInlineBlock(editor.document);
    if (firstBlock) {
      editor.setTextCursorPosition(firstBlock, "start");
    } else {
      editor.focus();
    }
  }, [editor, readOnly]);

  return (
    <div className={styles.pageShell}>
      <BlockNoteView
        editor={editor}
        editable={!readOnly}
        formattingToolbar={false}
        filePanel={false}
        renderEditor={false}
        onChange={(currentEditor) => onChange?.(currentEditor.document)}
      >
        {!readOnly && (
          <>
            <div className={styles.toolbarWrapper}>
              <FormattingToolbar>
                {getFormattingToolbarItems()}
              </FormattingToolbar>
            </div>
            <FilePanelController filePanel={uppyFilePanel} />
          </>
        )}
        <div
          className={styles.editorRoot}
          style={cssVars as CSSProperties}
          data-editable={!readOnly}
        >
          <BlockNoteViewEditor />
        </div>
      </BlockNoteView>
    </div>
  );
}
