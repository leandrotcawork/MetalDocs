import type { DocumentListItem } from "../../lib.types";
import { FillPage } from "../../features/documents/ck5/react/FillPage";

type ContentBuilderViewProps = {
  document: DocumentListItem | null;
  onBack: () => void;
  currentUserId?: string;
};

export function ContentBuilderView(props: ContentBuilderViewProps) {
  if (!props.document) {
    return (
      <section className="content-builder-empty">
        <strong>Nenhum documento selecionado.</strong>
        <p>Abra um documento antes de editar o conteudo.</p>
        <button type="button" className="ghost-button" onClick={props.onBack}>
          Voltar ao acervo
        </button>
      </section>
    );
  }

  if (!props.document.documentId.trim()) {
    return (
      <section className="content-builder-empty">
        <strong>Documento ainda nao persistido.</strong>
        <p>Crie o documento antes de abrir o editor.</p>
        <button type="button" className="ghost-button" onClick={props.onBack}>
          Voltar para criar documento
        </button>
      </section>
    );
  }

  void props.currentUserId;
  const templateId = props.document.documentType?.trim() || props.document.documentId;
  return <FillPage tplId={templateId} docId={props.document.documentId} />;
}
