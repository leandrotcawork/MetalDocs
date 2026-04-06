import type { DocumentListItem } from "../../lib.types";
import { BrowserDocumentEditorView } from "../../features/documents/browser-editor/BrowserDocumentEditorView";
import { LegacyContentBuilderView } from "./LegacyContentBuilderView";

type ContentBuilderViewProps = {
  document: DocumentListItem | null;
  onBack: () => void;
  onCreateFromDraft?: (contentDraft: Record<string, unknown>) => Promise<{ documentId: string; pdfUrl: string; version: number | null }>;
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

  const isPersistedDocument = props.document.documentId.trim().length > 0;
  const isBrowserEditorEligible = isPersistedDocument && props.document.documentProfile.trim().toLowerCase() === "po";

  if (isBrowserEditorEligible) {
    return <BrowserDocumentEditorView document={props.document} onBack={props.onBack} />;
  }

  return (
    <LegacyContentBuilderView
      document={props.document}
      onBack={props.onBack}
      onCreateFromDraft={props.onCreateFromDraft}
    />
  );
}
