export type TemplateAuthorPageProps = {
  templateId: string;
  versionNum: number;
  onNavigateToVersion?: (templateId: string, versionNum: number) => void;
};

export function TemplateAuthorPage({ templateId, versionNum }: TemplateAuthorPageProps) {
  return <div>Author: {templateId} v{versionNum} (placeholder)</div>;
}
