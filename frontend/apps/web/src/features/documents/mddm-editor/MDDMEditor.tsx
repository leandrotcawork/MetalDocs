import { type PartialBlock } from "@blocknote/core";
import { BlockNoteView } from "@blocknote/mantine";
import { useCreateBlockNote } from "@blocknote/react";
import "@blocknote/core/fonts/inter.css";
import "@blocknote/mantine/style.css";
import { mddmSchema } from "./schema";
import styles from "./MDDMEditor.module.css";

export type MDDMEditorProps = {
  initialContent?: PartialBlock[];
  onChange?: (blocks: unknown[]) => void;
  readOnly?: boolean;
};

export function MDDMEditor({
  initialContent,
  onChange,
  readOnly,
}: MDDMEditorProps) {
  const editor = useCreateBlockNote({
    schema: mddmSchema,
    initialContent: initialContent?.length ? initialContent : undefined,
  });

  return (
    <div className={styles.editorRoot}>
      <BlockNoteView
        editor={editor}
        editable={!readOnly}
        onChange={(currentEditor) => onChange?.(currentEditor.document)}
      />
    </div>
  );
}
