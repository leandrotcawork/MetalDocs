import { type PartialBlock } from "@blocknote/core";
import { BlockNoteView } from "@blocknote/mantine";
import { useCreateBlockNote } from "@blocknote/react";
import { useEffect, useMemo, type CSSProperties } from "react";
import "@blocknote/core/fonts/inter.css";
import "@blocknote/mantine/style.css";
import "./mddm-editor-global.css";
import { mddmSchema } from "./schema";
import styles from "./MDDMEditor.module.css";

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
};

export function MDDMEditor({
  initialContent,
  onChange,
  readOnly,
  theme,
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

  const themeStyle = useMemo(() => {
    if (!theme) {
      return undefined;
    }

    const vars: Record<string, string> = {};
    if (theme.accent) vars["--mddm-accent"] = theme.accent;
    if (theme.accentLight) vars["--mddm-accent-light"] = theme.accentLight;
    if (theme.accentDark) vars["--mddm-accent-dark"] = theme.accentDark;
    if (theme.accentBorder) vars["--mddm-accent-border"] = theme.accentBorder;

    return Object.keys(vars).length > 0 ? (vars as CSSProperties) : undefined;
  }, [theme]);

  return (
    <div className={styles.pageShell}>
      <div className={styles.editorRoot} style={themeStyle}>
        <BlockNoteView
          editor={editor}
          editable={!readOnly}
          onChange={(currentEditor) => onChange?.(currentEditor.document)}
        />
      </div>
    </div>
  );
}
