import type { SchemaSection } from "../contentSchemaTypes";
import { DynamicPreview } from "../../../features/documents/runtime/DynamicPreview";

type DocumentPreviewRendererProps = {
  sections: SchemaSection[];
  content: Record<string, unknown>;
  profileCode: string;
  documentCode: string;
  title: string;
  documentStatus: string;
  version: number | null;
  activeSectionKey?: string | null;
};

export function DocumentPreviewRenderer({
  sections,
  content,
  profileCode,
  documentCode,
  title,
  documentStatus,
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
      documentStatus={documentStatus}
      version={version}
      activeSectionKey={activeSectionKey}
    />
  );
}
