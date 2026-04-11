import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { exportDocumentDocx, getDocumentBrowserEditorBundle, saveDocumentBrowserContent } from "../../../api/documents";
import type { DocumentBrowserEditorBundleResponse, DocumentListItem, RendererPin } from "../../../lib.types";
import { formatDocumentDisplayName } from "../../shared/documentDisplay";
import { normalizeDocumentProfileCode } from "../../shared/documentProfile";
import { isMddmNativeExportEnabled } from "../../featureFlags";
import { exportDocx as mddmExportDocx } from "../mddm-editor/engine/export";
import styles from "./BrowserDocumentEditorView.module.css";
import { DocumentEditorHeader } from "./DocumentEditorHeader";
import { MDDMEditor, type MDDMTheme } from "../mddm-editor/MDDMEditor";
import { blockNoteToMDDM, mddmToBlockNote, type MDDMEnvelope } from "../mddm-editor/adapter";
import { SaveBeforeExportDialog } from "./SaveBeforeExportDialog";
import { runShadowExport } from "../mddm-editor/engine/shadow-testing/shadow-runner";
import { computeShadowDiff } from "../mddm-editor/engine/shadow-testing/shadow-diff";
import { postShadowDiff } from "../mddm-editor/engine/shadow-testing/shadow-telemetry";
import { unzipDocxDocumentXml } from "../mddm-editor/engine/golden/golden-helpers";

type BrowserDocumentEditorViewProps = {
  document: DocumentListItem;
  onBack: () => void;
};

type ViewState = "loading" | "ready" | "saving" | "error";

export function BrowserDocumentEditorView({ document, onBack }: BrowserDocumentEditorViewProps) {
  const [bundle, setBundle] = useState<DocumentBrowserEditorBundleResponse | null>(null);
  const [editorData, setEditorData] = useState("");
  const [blockNoteDocument, setBlockNoteDocument] = useState<unknown[] | null>(null);
  const [editorInstance, setEditorInstance] = useState(0);
  const [viewState, setViewState] = useState<ViewState>("loading");
  const [errorMessage, setErrorMessage] = useState("");
  const [errorCode, setErrorCode] = useState<"load" | "save" | "export" | "conflict" | null>(null);
  const [saveLabel, setSaveLabel] = useState("Nao salvo");
  const [isExporting, setIsExporting] = useState(false);
  const [exportDialogOpen, setExportDialogOpen] = useState(false);
  const [pendingExportKind, setPendingExportKind] = useState<"docx" | null>(null);
  const bundleRef = useRef<DocumentBrowserEditorBundleResponse | null>(null);
  const errorCodeRef = useRef<"load" | "save" | "export" | "conflict" | null>(null);

  useEffect(() => {
    bundleRef.current = bundle;
  }, [bundle]);

  useEffect(() => {
    errorCodeRef.current = errorCode;
  }, [errorCode]);

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
    setSaveLabel("Carregando...");

    try {
      const nextBundle = await getDocumentBrowserEditorBundle(document.documentId);
      if (activeRef?.cancelled) {
        return;
      }

      setBundle(nextBundle);
      setEditorData(nextBundle.body);

      try {
        const body = (nextBundle.body ?? "").trim();
        if (body && !body.startsWith("{")) {
          throw new Error("Unsupported legacy body format.");
        }
        let envelope: MDDMEnvelope;
        if (body) {
          envelope = JSON.parse(body) as MDDMEnvelope;
        } else if (
          nextBundle.templateSnapshot?.definition &&
          Array.isArray((nextBundle.templateSnapshot.definition as Record<string, unknown>).children) &&
          ((nextBundle.templateSnapshot.definition as Record<string, unknown>).children as unknown[]).length > 0
        ) {
          const def = nextBundle.templateSnapshot.definition as Record<string, unknown>;
          envelope = { mddm_version: 1, template_ref: null, blocks: def.children as MDDMEnvelope["blocks"] };
        } else {
          envelope = { mddm_version: 1, template_ref: null, blocks: [] };
        }
        setBlockNoteDocument(mddmToBlockNote(envelope) as unknown[]);
        setEditorInstance((current) => current + 1);

        setViewState("ready");
        setErrorCode(null);
        setErrorMessage("");
        setSaveLabel("Salvo");
      } catch {
        setBlockNoteDocument(null);
        setViewState("error");
        setErrorCode("load");
        setErrorMessage("Conteudo do documento em formato legado (nao MDDM). Este editor nao suporta abrir este formato.");
        setSaveLabel("Erro");
      }
    } catch {
      if (activeRef?.cancelled) {
        return;
      }
      const hadBundle = bundleRef.current !== null;
      const hadConflict = errorCodeRef.current === "conflict";
      setViewState("error");
      if (hadBundle && hadConflict) {
        setErrorCode("conflict");
        setErrorMessage("Nao foi possivel recarregar o documento. Verifique a conexao e tente novamente.");
        setSaveLabel("Conflito de rascunho");
        return;
      }
      setBundle(null);
      setEditorData("");
      setBlockNoteDocument(null);
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
  const documentProfileCode = normalizeDocumentProfileCode(document.documentProfile);
  const editorTheme = useMemo((): MDDMTheme | undefined => {
    const theme = bundle?.templateSnapshot?.definition?.theme;
    if (!theme) {
      return undefined;
    }

    return {
      accent: theme.accent,
      accentLight: theme.accentLight,
      accentDark: theme.accentDark,
      accentBorder: theme.accentBorder,
    };
  }, [bundle?.templateSnapshot?.definition?.theme]);

  const isReleased = document.status === "PUBLISHED";
  const isDirty = bundle !== null && editorData !== bundle.body;
  const isSaving = viewState === "saving";
  const latestVersion = bundle && bundle.versions.length > 0 ? bundle.versions[bundle.versions.length - 1] : null;
  const hasConflict = errorCode === "conflict";

  const rendererPin = useMemo(() => {
    if (!latestVersion) return null;
    return (latestVersion.renderer_pin as RendererPin | null | undefined) ?? null;
  }, [latestVersion]);

  async function handleSave(): Promise<boolean> {
    if (!bundle || isSaving || viewState !== "ready" || !document.documentId.trim()) {
      return false;
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
      return true;
    } catch (error) {
      setViewState("error");
      if (statusOf(error) === 409) {
        setErrorCode("conflict");
        setErrorMessage("O rascunho ficou desatualizado. Recarregue o documento para sincronizar a ultima revisao antes de salvar novamente.");
        setSaveLabel("Conflito de rascunho");
        return false;
      }
      setErrorCode("save");
      setErrorMessage("Nao foi possivel salvar o rascunho no editor do navegador.");
      setSaveLabel("Erro ao salvar");
      return false;
    }
  }

  function triggerBlobDownload(blob: Blob, filename: string) {
    const url = window.URL.createObjectURL(blob);
    const link = window.document.createElement("a");
    link.href = url;
    link.download = filename;
    window.document.body.appendChild(link);
    link.click();
    link.remove();
    window.setTimeout(() => window.URL.revokeObjectURL(url), 100);
  }

  async function runDocxExport(source: "live" | "saved" = "live") {
    const safeCode = (document.documentCode || "documento").trim().replace(/[^\w.-]+/g, "-");

    const exportStart = performance.now();
    setIsExporting(true);
    let legacyBlob: Blob | null = null;
    try {
      if (isMddmNativeExportEnabled("")) {
        const rawBody = source === "saved" ? (bundle?.body ?? "") : (editorData ?? "");
        const body = rawBody.trim();
        if (body && !body.startsWith("{")) {
          throw new Error("Document body is not in MDDM JSON format");
        }
        const envelope: MDDMEnvelope = body
          ? (JSON.parse(body) as MDDMEnvelope)
          : { mddm_version: 1, template_ref: null, blocks: [] };
        const blob = await mddmExportDocx(envelope, { rendererPin });
        triggerBlobDownload(blob, `${safeCode}.docx`);
      } else {
        const blob = await exportDocumentDocx(document.documentId);
        legacyBlob = blob;
        triggerBlobDownload(blob, `${safeCode}.docx`);
      }
      setErrorCode(null);
      setErrorMessage("");
    } catch (error) {
      setErrorCode("export");
      setErrorMessage("Nao foi possivel exportar o DOCX deste documento.");
      const status = statusOf(error);
      if (status === 503) {
        setErrorMessage("Servico de render indisponivel. Inicie o docgen e tente novamente.");
      }
    } finally {
      setIsExporting(false);
    }

    // Fire-and-forget shadow run AFTER the user-visible export completes (after finally).
    if (!isMddmNativeExportEnabled("") && legacyBlob !== null && bundle !== null) {
      const currentDurationMs = Math.round(performance.now() - exportStart);
      const rawBody = source === "saved" ? (bundle.body ?? "") : (editorData ?? "");
      const body = rawBody.trim();
      const envelope: MDDMEnvelope = body && body.startsWith("{")
        ? (JSON.parse(body) as MDDMEnvelope)
        : { mddm_version: 1, template_ref: null, blocks: [] };
      void runShadowAndReport({
        envelope,
        rendererPin,
        currentBlob: legacyBlob,
        currentDurationMs,
        documentId: document.documentId,
        versionNumber: latestVersion?.version ?? 0,
        userIdHash: "",
      });
    }
  }

  async function handleExportDocx() {
    if (!document.documentId.trim() || isExporting) {
      return;
    }

    if (!isMddmNativeExportEnabled("")) {
      await runDocxExport();
      return;
    }

    if (isDirty) {
      setPendingExportKind("docx");
      setExportDialogOpen(true);
      return;
    }

    await runDocxExport();
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

        <div className={styles.actions}>
          <button
            type="button"
            className={styles.exportButton}
            onClick={() => void handleExportDocx()}
            disabled={!bundle || isSaving || isExporting}
          >
            {isExporting ? "Exportando..." : "Exportar DOCX"}
          </button>
          <button
            type="button"
            className={styles.saveButton}
            onClick={() => void handleSave()}
            disabled={!bundle || viewState !== "ready" || isSaving || !isDirty || hasConflict}
          >
            Salvar rascunho
          </button>
        </div>
      </header>

      <div className={styles.metaBar}>
        <span className={styles.metaItem}>
          <span className={styles.metaLabel}>Profile</span>
          <strong>{documentProfileCode.toUpperCase()}</strong>
        </span>
        <span className={styles.metaItem}>
          <span className={styles.metaLabel}>Versao</span>
          <strong>{latestVersion?.version ?? "-"}</strong>
        </span>
        <span className={styles.metaItem}>
          <span className={styles.metaLabel}>Template</span>
          <strong>{bundle?.templateSnapshot?.templateKey ?? "-"}</strong>
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
                {errorCode !== "load" && errorCode !== "conflict" && canRetrySave ? (
                  <button type="button" className={styles.errorActionButtonSecondary} onClick={() => void handleSave()}>
                    Tentar salvar novamente
                  </button>
                ) : null}
              </div>
            </div>
          ) : null}
          <DocumentEditorHeader bundle={bundle} />
          <div className={styles.editorShell}>
            {blockNoteDocument ? (
              <MDDMEditor
                key={`${document.documentId}:${editorInstance}`}
                initialContent={blockNoteDocument as any}
                onChange={(blocks) => {
                  try {
                    const envelope = blockNoteToMDDM(blocks as any[]);
                    const nextData = JSON.stringify(envelope);

                    setEditorData(nextData);
                    if (viewState !== "saving") {
                      setViewState("ready");
                    }
                    if (errorCode !== null && errorCode !== "conflict") {
                      setErrorCode(null);
                      setErrorMessage("");
                    }
                    setSaveLabel(nextData === bundle.body ? "Salvo" : "Editando...");
                  } catch {
                    // Keep the editor responsive; surface an actionable error and avoid persisting invalid JSON.
                    if (viewState !== "saving") {
                      setViewState("ready");
                    }
                    setErrorCode("save");
                    setErrorMessage(
                      "Falha ao converter o conteudo do editor para o formato MDDM. Continue editando e tente salvar novamente. Se o erro persistir, recarregue o documento.",
                    );
                    setSaveLabel("Erro de conversao");
                  }
                }}
                theme={editorTheme}
              />
            ) : (
              <div className={styles.stateCard} role="alert">
                <strong>Conteudo indisponivel</strong>
                <p className={styles.errorText}>
                  {errorMessage || "Este documento nao possui conteudo MDDM valido para edicao."}
                </p>
              </div>
            )}
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

      <SaveBeforeExportDialog
        open={exportDialogOpen}
        isReleased={isReleased}
        onCancel={() => {
          setExportDialogOpen(false);
          setPendingExportKind(null);
        }}
        onSaveAndExport={async () => {
          setExportDialogOpen(false);
          const saved = await handleSave();
          if (saved && pendingExportKind === "docx") {
            await runDocxExport("saved");
          }
          setPendingExportKind(null);
        }}
        onExportSaved={async () => {
          setExportDialogOpen(false);
          if (pendingExportKind === "docx") {
            await runDocxExport("saved");
          }
          setPendingExportKind(null);
        }}
      />
    </section>
  );
}

function statusOf(error: unknown): number | undefined {
  if (error && typeof error === "object" && "status" in error && typeof (error as { status?: unknown }).status === "number") {
    return (error as { status: number }).status;
  }
  return undefined;
}

async function runShadowAndReport(input: {
  envelope: MDDMEnvelope;
  rendererPin: RendererPin | null;
  currentBlob: Blob;
  currentDurationMs: number;
  documentId: string;
  versionNumber: number;
  userIdHash: string;
}) {
  try {
    const [currentXml, shadow] = await Promise.all([
      unzipDocxDocumentXml(input.currentBlob),
      runShadowExport(input.envelope, input.rendererPin),
    ]);

    if (!shadow.ok) {
      void postShadowDiff({
        document_id: input.documentId,
        version_number: input.versionNumber,
        user_id_hash: input.userIdHash,
        current_xml_hash: "",
        shadow_xml_hash: "",
        diff_summary: { identical: false, shadow_failed: true },
        current_duration_ms: input.currentDurationMs,
        shadow_duration_ms: shadow.durationMs,
        shadow_error: shadow.error,
      });
      return;
    }

    const diff = computeShadowDiff(currentXml, shadow.xml);
    void postShadowDiff({
      document_id: input.documentId,
      version_number: input.versionNumber,
      user_id_hash: input.userIdHash,
      current_xml_hash: diff.current_xml_hash,
      shadow_xml_hash: diff.shadow_xml_hash,
      diff_summary: diff.diff_summary,
      current_duration_ms: input.currentDurationMs,
      shadow_duration_ms: shadow.durationMs,
    });
  } catch (err) {
    // Never surface shadow errors to the user.
    console.warn("shadow run failed", err);
  }
}
