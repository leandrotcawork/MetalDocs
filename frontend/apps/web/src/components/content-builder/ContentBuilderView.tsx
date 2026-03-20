import { useEffect, useMemo, useState } from "react";
import { api } from "../../lib.api";
import type { DocumentListItem, DocumentProfileSchemaItem } from "../../lib.types";
import { PdfPreview } from "../create/widgets/PdfPreview";

type ContentBuilderViewProps = {
  document: DocumentListItem | null;
  onBack: () => void;
};

type BuilderStatus = "loading" | "idle" | "dirty" | "saving" | "error";

export function ContentBuilderView(props: ContentBuilderViewProps) {
  const documentId = props.document?.documentId ?? "";
  const [contentDraft, setContentDraft] = useState<Record<string, unknown>>({});
  const [status, setStatus] = useState<BuilderStatus>("loading");
  const [error, setError] = useState("");
  const [pdfUrl, setPdfUrl] = useState("");
  const [version, setVersion] = useState<number | null>(null);
  const [previewCollapsed, setPreviewCollapsed] = useState(false);
  const [schema, setSchema] = useState<DocumentProfileSchemaItem | null>(null);

  const documentCode = useMemo(() => {
    if (!props.document?.documentId) return "--";
    return props.document.documentId.slice(0, 8).toUpperCase();
  }, [props.document?.documentId]);

  useEffect(() => {
    if (!documentId) {
      setStatus("idle");
      setContentDraft({});
      return;
    }
    let isActive = true;
    async function loadContent() {
      setStatus("loading");
      setError("");
      try {
        const [contentResponse, schemasResponse] = await Promise.all([
          api.getDocumentContentNative(documentId),
          props.document?.documentProfile
            ? api.listDocumentProfileSchemas(props.document.documentProfile)
            : Promise.resolve({ items: [] as DocumentProfileSchemaItem[] }),
        ]);
        if (!isActive) return;
        const items = Array.isArray(schemasResponse.items) ? schemasResponse.items : [];
        const activeSchema = items.find((item) => item.isActive) ?? items[0] ?? null;
        setSchema(activeSchema);
        setVersion(contentResponse.version);
        setContentDraft((contentResponse.content ?? {}) as Record<string, unknown>);
        setStatus("idle");
      } catch (err) {
        if (!isActive) return;
        if (statusOf(err) === 404) {
          setContentDraft({});
          setStatus("idle");
          return;
        }
        setError("Falha ao carregar o conteudo nativo.");
        setStatus("error");
      }
    }
    void loadContent();
    return () => {
      isActive = false;
    };
  }, [documentId]);

  async function handleSave() {
    if (!documentId) return;
    setError("");
    const parsedContent: Record<string, unknown> = contentDraft ?? {};
    setStatus("saving");
    try {
      const response = await api.saveDocumentContentNative(documentId, { content: parsedContent });
      setVersion(response.version);
      setPdfUrl(response.pdfUrl);
      setStatus("idle");
    } catch {
      setError("Falha ao salvar o conteudo.");
      setStatus("error");
    }
  }

  async function handleRenderPdf() {
    if (!documentId) return;
    if (status === "dirty") {
      await handleSave();
      return;
    }
    setError("");
    setStatus("saving");
    try {
      const response = await api.renderDocumentContentPdf(documentId);
      setPdfUrl(response.pdfUrl);
      setStatus("idle");
    } catch {
      setError("Nao foi possivel gerar o PDF.");
      setStatus("error");
    }
  }

  const statusLabel = status === "dirty"
    ? "Nao salvo"
    : status === "saving"
      ? "Salvando..."
      : "Salvo";

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

  return (
    <section className="content-builder">
      <header className="content-builder-header">
        <div>
          <div className="content-builder-code">{documentCode}</div>
          <h2 className="content-builder-title">{props.document.title}</h2>
          <div className="content-builder-meta">
            <span>Profile: {props.document.documentProfile.toUpperCase()}</span>
            <span>Status: {props.document.status}</span>
          </div>
        </div>
        <div className="content-builder-header-actions">
          <span className={`content-builder-status ${status === "dirty" ? "is-warning" : ""}`}>{statusLabel}</span>
          <button type="button" className="ghost-button" onClick={props.onBack}>
            Voltar
          </button>
        </div>
      </header>

      <div className="content-builder-body">
        <div className="content-builder-editor">
          <ContentSchemaForm
            schema={schema}
            value={contentDraft}
            onChange={(next) => {
              setContentDraft(next);
              setStatus((current) => (current === "dirty" ? current : "dirty"));
            }}
          />
          {error && <div className="content-builder-error">{error}</div>}
        </div>

        <aside className={`content-builder-preview ${previewCollapsed ? "is-collapsed" : ""}`}>
          {!previewCollapsed && (
            <div className="content-builder-preview-inner">
              <div className="content-builder-preview-header">
                <strong>Preview do PDF</strong>
                <button type="button" className="ghost-button" onClick={() => setPreviewCollapsed(true)}>
                  Recolher
                </button>
              </div>
              {pdfUrl ? (
                <PdfPreview url={pdfUrl} className="content-builder-preview-frame" width={300} />
              ) : (
                <div className="content-builder-preview-empty">
                  <strong>Nenhum PDF gerado</strong>
                  <span>Salve o conteudo e clique em "Gerar PDF" para visualizar.</span>
                </div>
              )}
            </div>
          )}
          {previewCollapsed && (
            <button type="button" className="content-builder-preview-collapsed" onClick={() => setPreviewCollapsed(false)}>
              Preview PDF
            </button>
          )}
        </aside>
      </div>

      <footer className="content-builder-footer">
        <span>Versao: {version ?? "-"}</span>
        <div className="content-builder-footer-actions">
          <button type="button" className="ghost-button" onClick={handleSave} disabled={status === "saving" || status === "loading"}>
            Salvar
          </button>
          <button type="button" onClick={handleRenderPdf} disabled={status === "saving" || status === "loading"}>
            Gerar PDF
          </button>
        </div>
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

type ContentSchemaFormProps = {
  schema: DocumentProfileSchemaItem | null;
  value: Record<string, unknown>;
  onChange: (next: Record<string, unknown>) => void;
};

type SchemaSection = {
  key: string;
  title?: string;
  description?: string;
  fields?: SchemaField[];
};

type SchemaField = {
  key: string;
  label?: string;
  type?: string;
  required?: boolean;
  options?: string[];
  itemType?: string;
  columns?: SchemaField[];
};

function ContentSchemaForm(props: ContentSchemaFormProps) {
  const schema = props.schema?.contentSchema as { sections?: SchemaSection[] } | undefined;
  const sections = Array.isArray(schema?.sections) ? schema?.sections : [];

  if (!props.schema) {
    return (
      <div className="content-builder-section">
        <div className="content-builder-section-head">
          <strong>Conteudo estruturado</strong>
          <small>Schema nao disponivel para este profile.</small>
        </div>
        <div className="content-builder-empty">Sem schema ativo.</div>
      </div>
    );
  }

  return (
    <>
      {sections.map((section) => (
        <ContentSection
          key={section.key}
          section={section}
          value={props.value}
          onChange={props.onChange}
        />
      ))}
    </>
  );
}

type ContentSectionProps = {
  section: SchemaSection;
  value: Record<string, unknown>;
  onChange: (next: Record<string, unknown>) => void;
};

function ContentSection(props: ContentSectionProps) {
  const { section } = props;
  const sectionKey = section.key;
  const sectionValue = (props.value[sectionKey] as Record<string, unknown>) ?? {};

  function updateSectionField(fieldKey: string, nextValue: unknown) {
    const nextSection = { ...sectionValue, [fieldKey]: nextValue };
    props.onChange({ ...props.value, [sectionKey]: nextSection });
  }

  return (
    <div className="content-builder-section">
      <div className="content-builder-section-head">
        <strong>{section.title ?? section.key}</strong>
        {section.description && <small>{section.description}</small>}
      </div>
      <div className="content-builder-section-body">
        {(section.fields ?? []).map((field) => (
          <SchemaFieldRenderer
            key={`${sectionKey}-${field.key}`}
            field={field}
            value={sectionValue[field.key]}
            onChange={(next) => updateSectionField(field.key, next)}
          />
        ))}
      </div>
    </div>
  );
}

type SchemaFieldRendererProps = {
  field: SchemaField;
  value: unknown;
  onChange: (next: unknown) => void;
};

function SchemaFieldRenderer(props: SchemaFieldRendererProps) {
  const fieldType = props.field.type ?? "text";
  if (fieldType === "textarea") {
    return (
      <label className="content-builder-field">
        <span>{props.field.label ?? props.field.key}</span>
        <textarea
          value={(props.value as string) ?? ""}
          onChange={(event) => props.onChange(event.target.value)}
          rows={4}
        />
      </label>
    );
  }
  if (fieldType === "select") {
    return (
      <label className="content-builder-field">
        <span>{props.field.label ?? props.field.key}</span>
        <select value={(props.value as string) ?? ""} onChange={(event) => props.onChange(event.target.value)}>
          <option value="">Selecione</option>
          {(props.field.options ?? []).map((option) => (
            <option key={option} value={option}>{option}</option>
          ))}
        </select>
      </label>
    );
  }
  if (fieldType === "number") {
    return (
      <label className="content-builder-field">
        <span>{props.field.label ?? props.field.key}</span>
        <input
          type="number"
          value={props.value as number | string | undefined || ""}
          onChange={(event) => props.onChange(event.target.value === "" ? "" : Number(event.target.value))}
        />
      </label>
    );
  }
  if (fieldType === "array") {
    const items = Array.isArray(props.value) ? props.value : [];
    return (
      <div className="content-builder-field">
        <span>{props.field.label ?? props.field.key}</span>
        <div className="content-builder-array">
          {items.map((item, index) => (
            <div key={`${props.field.key}-${index}`} className="content-builder-array-row">
              <input
                value={item as string}
                onChange={(event) => {
                  const next = [...items];
                  next[index] = event.target.value;
                  props.onChange(next);
                }}
              />
              <button
                type="button"
                className="ghost-button"
                onClick={() => props.onChange(items.filter((_, itemIndex) => itemIndex !== index))}
              >
                Remover
              </button>
            </div>
          ))}
          <button
            type="button"
            className="ghost-button"
            onClick={() => props.onChange([...items, ""])}
          >
            Adicionar item
          </button>
        </div>
      </div>
    );
  }
  if (fieldType === "table") {
    const rows = Array.isArray(props.value) ? props.value : [];
    const columns = props.field.columns ?? [];
    return (
      <div className="content-builder-field">
        <span>{props.field.label ?? props.field.key}</span>
        <div className="content-builder-table">
          <div className="content-builder-table-head">
            {columns.map((column) => (
              <span key={column.key}>{column.label ?? column.key}</span>
            ))}
            <span />
          </div>
          {rows.map((row, rowIndex) => (
            <div key={`${props.field.key}-${rowIndex}`} className="content-builder-table-row">
              {columns.map((column) => (
                <input
                  key={`${props.field.key}-${rowIndex}-${column.key}`}
                  value={(row as Record<string, unknown>)?.[column.key] as string ?? ""}
                  onChange={(event) => {
                    const nextRows = [...rows];
                    const nextRow = { ...(rows[rowIndex] as Record<string, unknown>), [column.key]: event.target.value };
                    nextRows[rowIndex] = nextRow;
                    props.onChange(nextRows);
                  }}
                />
              ))}
              <button
                type="button"
                className="ghost-button"
                onClick={() => props.onChange(rows.filter((_, idx) => idx !== rowIndex))}
              >
                Remover
              </button>
            </div>
          ))}
          <button
            type="button"
            className="ghost-button"
            onClick={() => props.onChange([...rows, {}])}
          >
            Adicionar linha
          </button>
        </div>
      </div>
    );
  }
  return (
    <label className="content-builder-field">
      <span>{props.field.label ?? props.field.key}</span>
      <input
        value={(props.value as string) ?? ""}
        onChange={(event) => props.onChange(event.target.value)}
      />
    </label>
  );
}
