import { CKEditor } from "@ckeditor/ckeditor5-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { DecoupledEditor } from "ckeditor5";
import "ckeditor5/ckeditor5.css";

import { getDocumentBrowserEditorBundle, saveDocumentBrowserContent } from "../../../api/documents";
import type { DocumentBrowserEditorBundleResponse, DocumentListItem } from "../../../lib.types";
import { formatDocumentDisplayName } from "../../shared/documentDisplay";
import { browserDocumentEditorClass, browserDocumentEditorConfig } from "./ckeditorConfig";
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
  const [errorCode, setErrorCode] = useState<"load" | "save" | "conflict" | null>(null);
  const [saveLabel, setSaveLabel] = useState("Nao salvo");
  const toolbarHostRef = useRef<HTMLDivElement | null>(null);

  const loadBundle = useCallback(async (activeRef?: { cancelled: boolean }) => {
    if (!document.documentId.trim()) {
      setBundle(null);
      setEditorData("");
      setViewState("error");
      setErrorCode("load");
      setErrorMessage("Este fluxo aceita apenas documentos ja persistidos. Abra um documento salvo pelo acervo.");
      setSaveLabel("Nao salvo");
      return;
    }

    setViewState("loading");
    setErrorCode(null);
    setErrorMessage("");
    setSaveLabel("Carregando...");

    try {
      const nextBundle = await getDocumentBrowserEditorBundle(document.documentId);
      if (activeRef?.cancelled) {
        return;
      }
      setBundle(nextBundle);
      setEditorData(nextBundle.body);
      setViewState("ready");
      setErrorCode(null);
      setErrorMessage("");
      setSaveLabel("Salvo");
    } catch {
      if (activeRef?.cancelled) {
        return;
      }
      setBundle(null);
      setEditorData("");
      setViewState("error");
      setErrorCode("load");
      setErrorMessage("Nao foi possivel carregar o bundle do editor do documento.");
      setSaveLabel("Erro");
    }
  }, [document.documentId]);

  useEffect(() => {
    const activeRef = { cancelled: false };
    void loadBundle(activeRef);
    return () => {
      activeRef.cancelled = true;
    };
  }, [loadBundle]);

  const documentTitle = useMemo(
    () => document.documentCode ?? formatDocumentDisplayName(document),
    [document],
  );

  const isDirty = bundle !== null && editorData !== bundle.body;
  const isSaving = viewState === "saving";
  const latestVersion = bundle && bundle.versions.length > 0 ? bundle.versions[bundle.versions.length - 1] : null;

  async function handleSave() {
    if (!bundle || isSaving || !document.documentId.trim()) {
      return;
    }

    setViewState("saving");
    setErrorCode(null);
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
      setErrorCode(null);
      setErrorMessage("");
      setSaveLabel("Salvo agora");
      window.setTimeout(() => {
        setSaveLabel((current) => (current === "Salvo agora" ? "Salvo ha pouco" : current));
      }, 3000);
    } catch (error) {
      setViewState("error");
      if (statusOf(error) === 409) {
        setErrorCode("conflict");
        setErrorMessage("O rascunho ficou desatualizado. Recarregue o documento para sincronizar a ultima revisao antes de salvar novamente.");
        setSaveLabel("Conflito de rascunho");
        return;
      }
      setErrorCode("save");
      setErrorMessage("Nao foi possivel salvar o rascunho no editor do navegador.");
      setSaveLabel("Erro ao salvar");
    }
  }

  const canRetrySave = Boolean(bundle) && !isSaving && isDirty;
  const showInlineError = errorMessage.trim().length > 0;

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
          {showInlineError ? (
            <div className={styles.errorBanner} role="alert">
              <div className={styles.errorCopy}>
                <strong>{errorCode === "conflict" ? "Conflito de rascunho" : "Falha no editor"}</strong>
                <p>{errorMessage}</p>
              </div>
              <div className={styles.errorActions}>
                <button type="button" className={styles.errorActionButton} onClick={() => void loadBundle()}>
                  Recarregar documento
                </button>
                {errorCode !== "load" && canRetrySave ? (
                  <button type="button" className={styles.errorActionButtonSecondary} onClick={() => void handleSave()}>
                    Tentar salvar novamente
                  </button>
                ) : null}
              </div>
            </div>
          ) : null}
          <div className={styles.toolbarShell} ref={toolbarHostRef} />
          <div className={styles.editorShell}>
            <CKEditor
              key={document.documentId}
              editor={browserDocumentEditorClass}
              config={browserDocumentEditorConfig}
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
                if (errorCode !== null) {
                  setErrorCode(null);
                  setErrorMessage("");
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
            <button type="button" className={styles.errorActionButton} onClick={() => void loadBundle()}>
              Recarregar documento
            </button>
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

function statusOf(error: unknown): number | undefined {
  if (error && typeof error === "object" && "status" in error && typeof (error as { status?: unknown }).status === "number") {
    return (error as { status: number }).status;
  }
  return undefined;
}
