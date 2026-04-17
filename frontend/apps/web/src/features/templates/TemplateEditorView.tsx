import { AuthorPage } from "../documents/ck5/react/AuthorPage";

type TemplateEditorViewProps = {
  profileCode: string;
  templateKey: string;
};

export function TemplateEditorView({ profileCode, templateKey }: TemplateEditorViewProps) {
  void profileCode;
  return <AuthorPage tplId={templateKey} />;
}
