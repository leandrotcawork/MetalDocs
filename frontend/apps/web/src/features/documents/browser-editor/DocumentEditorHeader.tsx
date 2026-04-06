import type { DocumentBrowserEditorBundleResponse } from "../../../lib.types";
import styles from "./DocumentEditorHeader.module.css";

type DocumentEditorHeaderProps = {
  bundle: DocumentBrowserEditorBundleResponse;
};

export function DocumentEditorHeader({ bundle }: DocumentEditorHeaderProps) {
  const { document, versions } = bundle;
  const latest = versions.length > 0 ? versions[versions.length - 1] : null;
  const revision = latest ? String(latest.version).padStart(2, "0") : "—";
  const createdAt = document.createdAt
    ? formatDate(document.createdAt)
    : latest?.createdAt
    ? formatDate(latest.createdAt)
    : "—";

  return (
    <div className={styles.header} data-testid="document-editor-header">
      <div className={styles.top}>
        <span>Metal Nobre</span>
        <span className={styles.code}>
          {document.documentCode ?? "—"} · Rev. {revision}
        </span>
      </div>
      <p className={styles.title}>{document.title}</p>
      <div className={styles.meta}>
        <MetaItem label="Tipo" value={document.documentType} />
        <MetaItem label="Elaborado por" value={document.ownerId} />
        <MetaItem label="Data" value={createdAt} />
        <MetaItem label="Status" value={document.status} />
        <MetaItem label="Aprovado por" value="—" />
      </div>
    </div>
  );
}

function MetaItem({ label, value }: { label: string; value: string }) {
  return (
    <span className={styles.metaItem}>
      <span className={styles.metaLabel}>{label}</span>
      <span className={styles.metaValue}>{value}</span>
    </span>
  );
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString("pt-BR", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
    });
  } catch {
    return "—";
  }
}
