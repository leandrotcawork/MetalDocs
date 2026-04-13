import type { PartialBlock } from "@blocknote/core";
import { MDDMEditor, type MDDMTheme } from "./MDDMEditor";

export type MDDMViewerProps = {
  initialContent?: PartialBlock[];
  theme?: MDDMTheme;
  documentId?: string;
};

export function MDDMViewer({ initialContent, theme, documentId }: MDDMViewerProps) {
  return (
    <MDDMEditor
      initialContent={initialContent}
      theme={theme}
      readOnly={true}
      documentId={documentId}
    />
  );
}
