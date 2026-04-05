import { CKEditor } from "@ckeditor/ckeditor5-react";
import { useEffect, useMemo, useRef, useState } from "react";
import type { DecoupledEditor } from "ckeditor5";
import "ckeditor5/ckeditor5.css";

import { getDocumentBrowserEditorBundle, saveDocumentBrowserContent } from "../../../api/documents";
import type { DocumentBrowserEditorBundleResponse, DocumentListItem } from "../../../lib.types";
import { formatDocumentDisplayName } from "../../shared/documentDisplay";
import { browserDocumentEditorClass, buildBrowserDocumentEditorConfig } from "./ckeditorConfig";
import styles from "./BrowserDocumentEditorView.module.css";

type BrowserDocumentEditorViewProps = {
  document: DocumentListItem;
  onBack: () => void;
};

type ViewState = "loading" | "ready" | "saving" | "error";

export function BrowserDocumentEditorView({ document, onBack }: BrowserDocumentEditorViewProps) {
  const [bundle, setBundle] = useState<DocumentBrowserEditorBundleResponse | null>(null);
  const [editorData, setEditorData] = useState("");
  const [viewState, setViewState] = useState<ViewState>("loading");
  const [errorMessage, setErrorMessage] = useState("");
  const [saveLabel, setSaveLabel] = useState("Nao salvo");
  const toolbarHostRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function loadBundle() {
      if (!document.documentId.trim()) {
        setBundle(null);
        setEditorData("");
        setViewState("error");
        setErrorMessage("Este fluxo aceita apenas documentos ja persistidos. Abra um documento salvo pelo acervo.");
        setSaveLabel("Nao salvo");
        return;
      }

      setViewState("loading");
      setErrorMessage("");
      setSaveLabel("Carregando...");

      try {
        const nextBundle = await getDocumentBrowserEditorBundle(document.documentId);
        if (cancelled) {
          return;
        }
        setBundle(nextBundle);
        setEditorData(nextBundle.body);
        setViewState("ready");
        setSaveLabel("Salvo");
      } catch {
        if (cancelled) {
          return;
        }
        setBundle(null);
        setEditorData("");
        setViewState("error");
        setErrorMessage("Nao foi possivel carregar o bundle do editor do documento.");
        setSaveLabel("Erro");
      }
    }

    void loadBundle();

    return () => {
      cancelled = true;
    };
  }, [document.documentId]);

  const documentTitle = useMemo(
    () => document.documentCode ?? formatDocumentDisplayName(document),
    [document],
  );

  const editorConfig = useMemo(() => buildBrowserDocumentEditorConfig(editorData), [editorData]);
  const isDirty = bundle !== null && editorData !== bundle.body;
  const isSaving = viewState === "saving";
  const latestVersion = bundle && bundle.versions.length > 0 ? bundle.versions[bundle.versions.length - 1] : null;

  async function handleSave() {
    if (!bundle || isSaving || !document.documentId.trim()) {
      return;
    }

    setViewState("saving");
    setErrorMessage("");
    setSaveLabel("Salvando...");

    try {
      const response = await saveDocumentBrowserContent(document.documentId, {
        body: editorData,
        draftToken: bundle.draftToken,
      });

      setBundle((current) => {
        if (!current) {
          return current;
        }

        const nextVersions = current.versions.length === 0
          ? current.versions
          : current.versions.map((item, index, items) => (
              index === items.length - 1
                ? { ...item, version: response.version }
                : item
            ));

        return {
          ...current,
          body: editorData,
          draftToken: response.draftToken,
          versions: nextVersions,
        };
      });
      setViewState("ready");
      setSaveLabel("Salvo agora");
      window.setTimeout(() => {
        setSaveLabel((current) => (current === "Salvo agora" ? "Salvo ha pouco" : current));
      }, 3000);
    } catch {
      setViewState("error");
      setErrorMessage("Nao foi possivel salvar o rascunho no editor do navegador.");
      setSaveLabel("Erro ao salvar");
    }
  }

  return (
    <section className={styles.root} data-testid="browser-document-editor">
      <header className={styles.topbar}>
        <button type="button" className={styles.backButton} onClick={onBack}>
          Voltar
        </button>

        <div className={styles.documentMeta}>
          <p className={styles.breadcrumbs}>MetalDocs / Acervo / {documentTitle}</p>
          <div className={styles.titleRow}>
            <h2 className={styles.documentTitle}>{document.title || documentTitle}</h2>
            <span className={styles.statusPill}>
              <span className={styles.statusDot} aria-hidden="true" />
              {document.status}
            </span>
          </div>
        </div>

        <button
          type="button"
          className={styles.saveButton}
          onClick={handleSave}
          disabled={!bundle || isSaving || !isDirty}
        >
          Salvar rascunho
        </button>
      </header>

      <div className={styles.metaBar}>
        <span className={styles.metaItem}>
          <span className={styles.metaLabel}>Profile</span>
          <strong>{document.documentProfile.toUpperCase()}</strong>
        </span>
        <span className={styles.metaItem}>
          <span className={styles.metaLabel}>Versao</span>
          <strong>{latestVersion?.version ?? "-"}</strong>
        </span>
        <span className={styles.metaItem}>
          <span className={styles.metaLabel}>Template</span>
          <strong>{bundle?.templateSnapshot.templateKey ?? "-"}</strong>
        </span>
      </div>

      {bundle ? (
        <div className={styles.surface}>
          <div className={styles.toolbarShell} ref={toolbarHostRef} />
          <div className={styles.editorShell}>
            <CKEditor
              key={document.documentId}
              editor={browserDocumentEditorClass}
              config={editorConfig}
              data={bundle.body}
              onReady={(editor: DecoupledEditor) => {
                const toolbarElement = editor.ui.view.toolbar.element;
                const toolbarHost = toolbarHostRef.current;

                if (!toolbarHost || !toolbarElement) {
                  return;
                }

                toolbarHost.replaceChildren(toolbarElement);
              }}
              onAfterDestroy={() => {
                toolbarHostRef.current?.replaceChildren();
              }}
              onChange={(_, editor) => {
                const nextData = editor.getData();
                setEditorData(nextData);
                if (viewState !== "saving") {
                  setViewState("ready");
                }
                setSaveLabel(nextData === bundle.body ? "Salvo" : "Editando...");
              }}
            />
          </div>
        </div>
      ) : (
        <div className={styles.statePanel}>
          <div className={styles.stateCard}>
            <strong>{viewState === "loading" ? "Carregando editor" : "Editor indisponivel"}</strong>
            <p className={errorMessage ? styles.errorText : undefined}>
              {errorMessage || "Preparando o bundle do documento para o editor do navegador."}
            </p>
          </div>
        </div>
      )}

      <footer className={styles.footer}>
        <span className={styles.footerHint}>Superficie unica de edicao, sem painel legado de preview.</span>
        <span
          className={[
            styles.saveState,
            isSaving ? styles.saveStateBusy : "",
            isDirty ? styles.saveStatePending : "",
          ].filter(Boolean).join(" ")}
        >
          <span className={styles.saveStateDot} aria-hidden="true" />
          {saveLabel}
        </span>
      </footer>
    </section>
  );
}
