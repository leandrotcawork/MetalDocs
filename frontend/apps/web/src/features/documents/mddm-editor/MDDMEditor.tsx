import { type PartialBlock } from "@blocknote/core";
import { BlockNoteView } from "@blocknote/mantine";
import { useCreateBlockNote } from "@blocknote/react";
import { useEffect, useMemo, type CSSProperties } from "react";
import "@blocknote/core/fonts/inter.css";
import "@blocknote/mantine/style.css";
import "./mddm-editor-global.css";
import { mddmSchema } from "./schema";
import styles from "./MDDMEditor.module.css";
import { defaultLayoutTokens, tokensToCssVars } from "./engine/layout-ir";

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
};

export function MDDMEditor({
  initialContent,
  onChange,
  readOnly,
  theme,
  onEditorReady,
}: MDDMEditorProps) {
  const editor = useCreateBlockNote({
    schema: mddmSchema,
    initialContent: initialContent?.length ? initialContent : undefined,
  });

  useEffect(() => {
    if (import.meta.env.DEV) {
      (window as any).__mddmEditor = editor;
      return () => { delete (window as any).__mddmEditor; };
    }
    return undefined;
  }, [editor]);

  useEffect(() => {
    onEditorReady?.(editor);
  }, [editor, onEditorReady]);

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

  return (
    <div className={styles.pageShell}>
      <div
        className={styles.editorRoot}
        style={cssVars as CSSProperties}
        data-editable={!readOnly}
      >
        <BlockNoteView
          editor={editor}
          editable={!readOnly}
          onChange={(currentEditor) => onChange?.(currentEditor.document)}
        />
      </div>
    </div>
  );
}
