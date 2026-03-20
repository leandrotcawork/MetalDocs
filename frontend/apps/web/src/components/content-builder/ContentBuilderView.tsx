import { useEffect, useMemo, useReducer, useState } from "react";
import { api } from "../../lib.api";
import type { DocumentListItem, DocumentProfileSchemaItem } from "../../lib.types";
import { PdfPreview } from "../create/widgets/PdfPreview";

type ContentBuilderViewProps = {
  document: DocumentListItem | null;
  onBack: () => void;
};

type BuilderStatus = "loading" | "idle" | "dirty" | "saving" | "rendering" | "error";

type BuilderState = {
  status: BuilderStatus;
  error: string;
  pdfUrl: string;
  version: number | null;
  contentDraft: Record<string, unknown>;
  schema: DocumentProfileSchemaItem | null;
  previewCollapsed: boolean;
};

type BuilderAction =
  | { type: "load_start" }
  | { type: "load_success"; payload: { contentDraft: Record<string, unknown>; schema: DocumentProfileSchemaItem | null; version: number | null; pdfUrl: string } }
  | { type: "load_error"; payload: { message: string } }
  | { type: "set_draft"; payload: { contentDraft: Record<string, unknown> } }
  | { type: "set_status"; payload: { status: BuilderStatus } }
  | { type: "set_error"; payload: { message: string } }
  | { type: "set_pdf"; payload: { pdfUrl: string } }
  | { type: "set_preview"; payload: { collapsed: boolean } };

const initialState: BuilderState = {
  status: "loading",
  error: "",
  pdfUrl: "",
  version: null,
  contentDraft: {},
  schema: null,
  previewCollapsed: false,
};

function reducer(state: BuilderState, action: BuilderAction): BuilderState {
  switch (action.type) {
    case "load_start":
      return { ...state, status: "loading", error: "" };
    case "load_success":
      return {
        ...state,
        status: "idle",
        error: "",
        contentDraft: action.payload.contentDraft,
        schema: action.payload.schema,
        version: action.payload.version,
        pdfUrl: action.payload.pdfUrl,
      };
    case "load_error":
      return { ...state, status: "error", error: action.payload.message };
    case "set_draft":
      return { ...state, contentDraft: action.payload.contentDraft };
    case "set_status":
      return { ...state, status: action.payload.status };
    case "set_error":
      return { ...state, error: action.payload.message };
    case "set_pdf":
      return { ...state, pdfUrl: action.payload.pdfUrl };
    case "set_preview":
      return { ...state, previewCollapsed: action.payload.collapsed };
    default:
      return state;
  }
}

export function ContentBuilderView(props: ContentBuilderViewProps) {
  const documentId = props.document?.documentId ?? "";
  const [state, dispatch] = useReducer(reducer, initialState);
  const { status, error, pdfUrl, version, contentDraft, schema, previewCollapsed } = state;

  const documentCode = useMemo(() => {
    if (!props.document?.documentId) return "--";
    return props.document.documentId.slice(0, 8).toUpperCase();
  }, [props.document?.documentId]);

  useEffect(() => {
    if (!documentId) {
      dispatch({ type: "set_status", payload: { status: "idle" } });
      dispatch({ type: "set_draft", payload: { contentDraft: {} } });
      return;
    }
    let isActive = true;
    async function loadContent() {
      dispatch({ type: "load_start" });
      try {
        const [contentResponse, schemasResponse, pdfResponse] = await Promise.all([
          api.getDocumentContentNative(documentId),
          props.document?.documentProfile
            ? api.listDocumentProfileSchemas(props.document.documentProfile)
            : Promise.resolve({ items: [] as DocumentProfileSchemaItem[] }),
          api.getDocumentContentPdf(documentId).catch((err) => {
            if (statusOf(err) === 404) {
              return null;
            }
            throw err;
          }),
        ]);
        if (!isActive) return;
        const items = Array.isArray(schemasResponse.items) ? schemasResponse.items : [];
        const activeSchema = items.find((item) => item.isActive) ?? items[0] ?? null;
        dispatch({
          type: "load_success",
          payload: {
            contentDraft: (contentResponse.content ?? {}) as Record<string, unknown>,
            schema: activeSchema,
            version: contentResponse.version ?? null,
            pdfUrl: pdfResponse?.url ?? "",
          },
        });
      } catch (err) {
        if (!isActive) return;
        if (statusOf(err) === 404) {
          dispatch({
            type: "load_success",
            payload: { contentDraft: {}, schema: null, version: null, pdfUrl: "" },
          });
          return;
        }
        dispatch({ type: "load_error", payload: { message: "Falha ao carregar o conteudo nativo." } });
      }
    }
    void loadContent();
    return () => {
      isActive = false;
    };
  }, [documentId, props.document?.documentProfile]);

  async function handleSave() {
    if (!documentId) return;
    dispatch({ type: "set_error", payload: { message: "" } });
    const parsedContent: Record<string, unknown> = contentDraft ?? {};
    dispatch({ type: "set_status", payload: { status: "saving" } });
    try {
      const response = await api.saveDocumentContentNative(documentId, { content: parsedContent });
      dispatch({ type: "set_pdf", payload: { pdfUrl: response.pdfUrl } });
      dispatch({ type: "load_success", payload: { contentDraft: parsedContent, schema, version: response.version ?? null, pdfUrl: response.pdfUrl } });
    } catch {
      dispatch({ type: "load_error", payload: { message: "Falha ao salvar o conteudo." } });
    }
  }

  async function handleRenderPdf() {
    if (!documentId) return;
    if (status === "dirty") {
      await handleSave();
      return;
    }
    dispatch({ type: "set_error", payload: { message: "" } });
    dispatch({ type: "set_status", payload: { status: "rendering" } });
    try {
      const response = await api.renderDocumentContentPdf(documentId);
      dispatch({ type: "set_pdf", payload: { pdfUrl: response.pdfUrl } });
      dispatch({ type: "set_status", payload: { status: "idle" } });
    } catch {
      dispatch({ type: "load_error", payload: { message: "Nao foi possivel gerar o PDF." } });
    }
  }

  const statusLabel = status === "dirty"
    ? "Nao salvo"
    : status === "saving"
      ? "Salvando..."
      : status === "rendering"
        ? "Gerando PDF..."
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
          <div className="content-builder-breadcrumb">
            <span>MetalDocs</span>
            <span>/</span>
            <span>Acervo</span>
            <span>/</span>
            <strong>Editor</strong>
          </div>
          <div className="content-builder-code">{documentCode}</div>
          <h2 className="content-builder-title">{props.document.title}</h2>
          <div className="content-builder-meta">
            <span className="content-builder-badge">Profile: {props.document.documentProfile.toUpperCase()}</span>
            <span className="content-builder-badge">Status: {props.document.status}</span>
            <span className="content-builder-badge">Versao: {version ?? "-"}</span>
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
              dispatch({ type: "set_draft", payload: { contentDraft: next } });
              dispatch({ type: "set_status", payload: { status: "dirty" } });
            }}
          />
          {error && <div className="content-builder-error">{error}</div>}
        </div>

        <aside className={`content-builder-preview ${previewCollapsed ? "is-collapsed" : ""}`}>
          {!previewCollapsed && (
            <div className="content-builder-preview-inner">
              <div className="content-builder-preview-header">
                <strong>Preview do PDF</strong>
                <button type="button" className="ghost-button" onClick={() => dispatch({ type: "set_preview", payload: { collapsed: true } })}>
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
            <button type="button" className="content-builder-preview-collapsed" onClick={() => dispatch({ type: "set_preview", payload: { collapsed: false } })}>
              Preview PDF
            </button>
          )}
        </aside>
      </div>

      <footer className="content-builder-footer">
        <span>Versao ativa: {version ?? "-"}</span>
        <div className="content-builder-footer-actions">
          <button type="button" className="ghost-button" onClick={handleSave} disabled={status === "saving" || status === "loading" || status === "rendering"}>
            Salvar
          </button>
          <button type="button" onClick={handleRenderPdf} disabled={status === "saving" || status === "loading" || status === "rendering"}>
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
  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({});

  useEffect(() => {
    if (sections.length === 0) return;
    setExpandedSections((prev) => {
      let changed = false;
      const next = { ...prev };
      sections.forEach((section) => {
        if (!(section.key in next)) {
          next[section.key] = true;
          changed = true;
        }
      });
      return changed ? next : prev;
    });
  }, [sections]);

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
          expanded={expandedSections[section.key] ?? true}
          onToggle={() =>
            setExpandedSections((prev) => ({
              ...prev,
              [section.key]: !(prev[section.key] ?? true),
            }))
          }
        />
      ))}
    </>
  );
}

type ContentSectionProps = {
  section: SchemaSection;
  value: Record<string, unknown>;
  onChange: (next: Record<string, unknown>) => void;
  expanded: boolean;
  onToggle: () => void;
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
    <div className={`content-builder-section ${props.expanded ? "is-expanded" : "is-collapsed"}`}>
      <div className="content-builder-section-head">
        <button type="button" className="content-builder-section-toggle" onClick={props.onToggle}>
          <div className="content-builder-section-title">
            <strong>{section.title ?? section.key}</strong>
            {section.description && <small>{section.description}</small>}
          </div>
          <span className="content-builder-section-chevron" aria-hidden="true">
            {props.expanded ? "-" : "+"}
          </span>
        </button>
      </div>
      <div className="content-builder-section-body" aria-hidden={!props.expanded}>
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
  const label = props.field.label ?? props.field.key;
  const required = Boolean(props.field.required);
  if (fieldType === "textarea") {
    return (
      <label className="content-builder-field">
        <span>
          {label}
          {required && <em className="content-builder-required">*</em>}
        </span>
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
        <span>
          {label}
          {required && <em className="content-builder-required">*</em>}
        </span>
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
        <span>
          {label}
          {required && <em className="content-builder-required">*</em>}
        </span>
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
        <span>
          {label}
          {required && <em className="content-builder-required">*</em>}
        </span>
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
  if (fieldType === "checklist") {
    const items = normalizeChecklistItems(props.value);
    return (
      <div className="content-builder-field">
        <span>
          {label}
          {required && <em className="content-builder-required">*</em>}
        </span>
        <div className="content-builder-checklist">
          {items.map((item, index) => (
            <div key={`${props.field.key}-${index}`} className="content-builder-checklist-row">
              <input
                type="checkbox"
                checked={item.checked}
                onChange={(event) => {
                  const next = [...items];
                  next[index] = { ...item, checked: event.target.checked };
                  props.onChange(next);
                }}
              />
              <input
                value={item.label}
                onChange={(event) => {
                  const next = [...items];
                  next[index] = { ...item, label: event.target.value };
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
            onClick={() => props.onChange([...items, { label: "", checked: false }])}
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
        <span>
          {label}
          {required && <em className="content-builder-required">*</em>}
        </span>
        <div className="content-builder-table">
          <div className="content-builder-table-head">
            {columns.map((column) => (
              <span key={column.key}>{column.label ?? column.key}</span>
            ))}
            <span />
          </div>
          {rows.map((row, rowIndex) => (
            <div key={`${props.field.key}-${rowIndex}`} className="content-builder-table-row">
              {columns.map((column) => {
                const columnType = column.type ?? "text";
                const cellValue = (row as Record<string, unknown>)?.[column.key] as string ?? "";
                const handleCellChange = (nextValue: string | number) => {
                  const nextRows = [...rows];
                  const nextRow = { ...(rows[rowIndex] as Record<string, unknown>), [column.key]: nextValue };
                  nextRows[rowIndex] = nextRow;
                  props.onChange(nextRows);
                };
                if (columnType === "select") {
                  return (
                    <select
                      key={`${props.field.key}-${rowIndex}-${column.key}`}
                      value={cellValue}
                      onChange={(event) => handleCellChange(event.target.value)}
                    >
                      <option value="">Selecione</option>
                      {(column.options ?? []).map((option) => (
                        <option key={option} value={option}>{option}</option>
                      ))}
                    </select>
                  );
                }
                if (columnType === "number") {
                  return (
                    <input
                      key={`${props.field.key}-${rowIndex}-${column.key}`}
                      type="number"
                      value={cellValue}
                      onChange={(event) => handleCellChange(event.target.value === "" ? "" : Number(event.target.value))}
                    />
                  );
                }
                return (
                  <input
                    key={`${props.field.key}-${rowIndex}-${column.key}`}
                    value={cellValue}
                    onChange={(event) => handleCellChange(event.target.value)}
                  />
                );
              })}
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
      <span>
        {label}
        {required && <em className="content-builder-required">*</em>}
      </span>
      <input
        value={(props.value as string) ?? ""}
        onChange={(event) => props.onChange(event.target.value)}
      />
    </label>
  );
}

type ChecklistItem = { label: string; checked: boolean };

function normalizeChecklistItems(value: unknown): ChecklistItem[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.map((item) => {
    if (typeof item === "string") {
      return { label: item, checked: false };
    }
    if (item && typeof item === "object") {
      const typed = item as { label?: string; checked?: boolean };
      return { label: typed.label ?? "", checked: Boolean(typed.checked) };
    }
    return { label: "", checked: false };
  });
}
