import type { SchemaSection } from "../contentSchemaTypes";
import { DynamicPreview } from "../../../features/documents/runtime/DynamicPreview";

type DocumentPreviewRendererProps = {
  sections: SchemaSection[];
  content: Record<string, unknown>;
  profileCode: string;
  documentCode: string;
  title: string;
  version: number | null;
  activeSectionKey?: string | null;
};

export function DocumentPreviewRenderer({
  sections,
  content,
  profileCode,
  documentCode,
  title,
  version,
  activeSectionKey,
}: DocumentPreviewRendererProps) {
  return (
    <DynamicPreview
      schema={{ profileCode, version: version ?? 0, isActive: true, metadataRules: [], contentSchema: { sections } }}
      content={content}
      profileCode={profileCode}
      documentCode={documentCode}
      title={title}
      version={version}
      activeSectionKey={activeSectionKey}
    />
  );
}
