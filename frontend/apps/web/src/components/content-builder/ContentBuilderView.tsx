import type { DocumentListItem } from "../../lib.types";
import { BrowserDocumentEditorView } from "../../features/documents/browser-editor/BrowserDocumentEditorView";

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
  return <BrowserDocumentEditorView document={props.document} onBack={props.onBack} currentUserId={props.currentUserId} />;
}
