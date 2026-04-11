import type { PartialBlock } from "@blocknote/core";
import { MDDMEditor, type MDDMTheme } from "./MDDMEditor";

export type MDDMViewerProps = {
  initialContent?: PartialBlock[];
  theme?: MDDMTheme;
};

export function MDDMViewer({ initialContent, theme }: MDDMViewerProps) {
  return (
    <MDDMEditor
      initialContent={initialContent}
      theme={theme}
      readOnly={true}
    />
  );
}
