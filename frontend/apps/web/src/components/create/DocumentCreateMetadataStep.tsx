import { useEffect, useMemo, useRef, useState } from "react";
import type { DocumentProfileSchemaItem } from "../../lib.types";
import type { DocumentForm } from "./documentCreateTypes";
import { deleteMetadataField, parseMetadata, renameMetadataField, updateMetadataField } from "./documentCreateTypes";

type DocumentCreateMetadataStepProps = {
  form: DocumentForm;
  selectedProfileSchema: DocumentProfileSchemaItem | null;
  onDocumentFormChange: (next: DocumentForm) => void;
};

type CustomMetadataRow = {
  id: string;
  key: string;
  value: string;
};

export function DocumentCreateMetadataStep(props: DocumentCreateMetadataStepProps) {
  const metadataMap = parseMetadata(props.form.metadata);
  const metadataRules = (props.selectedProfileSchema?.metadataRules ?? []).filter((rule) => rule.name.trim().length > 0);
  const schemaNames = useMemo(() => new Set(metadataRules.map((rule) => rule.name)), [metadataRules]);
  const seedCustomRows = useMemo(() => {
    const rows = Object.entries(metadataMap)
      .filter(([key]) => key && !schemaNames.has(key))
      .map(([key, value]) => ({
        id: `custom-${key}`,
        key,
        value,
      }));
    return rows.length > 0 ? rows : [{ id: "custom-new-0", key: "", value: "" }];
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props.form.documentProfile, props.selectedProfileSchema?.version]);
  const idSeq = useRef(0);
  const [customRows, setCustomRows] = useState<CustomMetadataRow[]>(seedCustomRows);

  useEffect(() => {
    setCustomRows(seedCustomRows);
  }, [seedCustomRows]);

  function addCustomRow() {
    idSeq.current += 1;
    setCustomRows((current) => [
      ...current,
      { id: `custom-new-${idSeq.current}`, key: "", value: "" },
    ]);
  }

  function updateCustomRow(rowId: string, patch: Partial<CustomMetadataRow>) {
    setCustomRows((current) => current.map((row) => (row.id === rowId ? { ...row, ...patch } : row)));
  }

  function removeCustomRow(row: CustomMetadataRow) {
    setCustomRows((current) => current.filter((item) => item.id !== row.id));
    if (row.key.trim()) {
      props.onDocumentFormChange({ ...props.form, metadata: deleteMetadataField(props.form.metadata, row.key.trim()) });
    }
  }

  return (
    <div className="stack">
      <div className="metadata-editor" role="group" aria-label="Campos de metadata">
        <div className="metadata-editor-head" aria-hidden="true">
          <span>Variavel</span>
          <span>Valor</span>
          <span>Tipo</span>
          <button type="button" className="metadata-add" onClick={addCustomRow} aria-label="Adicionar campo">+</button>
        </div>

        <div className="metadata-editor-body">
          {metadataRules.map((rule) => (
            <div key={rule.name} className="metadata-row">
              <div className="metadata-key">
                <code>{rule.name}</code>
              </div>
              <input
                id={`metadata-${rule.name}`}
                value={metadataMap[rule.name] ?? ""}
                placeholder={rule.required ? "Obrigatorio" : "Opcional"}
                onChange={(event) => props.onDocumentFormChange({
                  ...props.form,
                  metadata: updateMetadataField(props.form.metadata, rule.name, event.target.value),
                })}
              />
              <span className="metadata-type">
                {rule.type}
                {rule.required ? <span className="metadata-required">Obrigatorio</span> : null}
              </span>
              <span />
            </div>
          ))}

          {customRows.map((row) => (
            <div key={row.id} className="metadata-row is-custom">
              <input
                aria-label="Variavel"
                placeholder="variavel"
                value={row.key}
                onChange={(event) => {
                  const nextKey = event.target.value;
                  updateCustomRow(row.id, { key: nextKey });
                  if (row.key.trim()) {
                    const nextMetadata = renameMetadataField(props.form.metadata, row.key.trim(), nextKey, row.value);
                    props.onDocumentFormChange({ ...props.form, metadata: nextMetadata });
                    return;
                  }
                  if (nextKey.trim()) {
                    props.onDocumentFormChange({ ...props.form, metadata: updateMetadataField(props.form.metadata, nextKey.trim(), row.value) });
                  }
                }}
              />
              <input
                aria-label="Valor"
                placeholder="valor"
                value={row.value}
                onChange={(event) => {
                  const nextValue = event.target.value;
                  updateCustomRow(row.id, { value: nextValue });
                  if (row.key.trim()) {
                    props.onDocumentFormChange({ ...props.form, metadata: updateMetadataField(props.form.metadata, row.key.trim(), nextValue) });
                  }
                }}
              />
              <span className="metadata-type">custom</span>
              <button type="button" className="metadata-remove" onClick={() => removeCustomRow(row)} aria-label="Remover campo">
                Remover
              </button>
            </div>
          ))}

          {metadataRules.length === 0 && customRows.length === 0 ? (
            <p className="catalog-muted">Nenhum campo de metadata configurado para este tipo.</p>
          ) : null}
        </div>

      </div>
    </div>
  );
}
