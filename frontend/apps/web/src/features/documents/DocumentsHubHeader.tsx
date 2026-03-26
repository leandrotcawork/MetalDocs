import { WorkspaceHeroHeader } from "../../components/ui/WorkspaceHeroHeader";

type DocumentsHubHeaderProps = {
  title: string;
  searchQuery: string;
  onSearchQueryChange: (value: string) => void;
  variant?: "default" | "compact";
};

export function DocumentsHubHeader(props: DocumentsHubHeaderProps) {
  return (
    <WorkspaceHeroHeader
      title={props.title}
      subtitle="Acervo organizado por areas, tipos e status. Navegue pelos documentos mais relevantes."
      searchQuery={props.searchQuery}
      onSearchQueryChange={props.onSearchQueryChange}
      variant={props.variant}
    />
  );
}
