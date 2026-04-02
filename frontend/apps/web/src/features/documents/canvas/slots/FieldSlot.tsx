import type { ChangeEvent } from "react";
import type { RuntimeDocumentSchema, RuntimeScalarField } from "../../runtime/schemaRuntimeTypes";
import styles from "../DocumentCanvas.module.css";
import { resolveCanvasSlotBinding } from "../slotBindings";
import { readCanvasSlotValue, writeCanvasSlotValue } from "../slotValues";

type FieldSlotProps = {
  path: string;
  schema: RuntimeDocumentSchema | null;
  values: Record<string, unknown>;
  onChange: (next: Record<string, unknown>) => void;
  readOnly?: boolean;
};

export function FieldSlot({ path, schema, values, onChange, readOnly }: FieldSlotProps) {
  const binding = resolveCanvasSlotBinding(schema, path);
  const field = binding?.field && binding.field.kind === "scalar" ? (binding.field as RuntimeScalarField) : null;
  const value = readCanvasSlotValue(values, path);
  const label = binding?.label ?? prettifyPath(path);
  const description = binding?.description ?? "";
  const required = Boolean(binding?.required);

  return (
    <div className={styles.slot}>
      <div className={styles.slotHeader}>
        <div className={styles.slotLabelRow}>
          <span className={styles.slotLabel}>{label}</span>
          {required ? <span className={styles.slotRequired}>*</span> : null}
        </div>
        <span className={styles.slotPath}>{binding?.sectionKey ?? path}</span>
      </div>
      {description ? <div className={styles.slotDescription}>{description}</div> : null}
      {renderControl(field, value, readOnly, (nextValue) => onChange(writeCanvasSlotValue(values, path, nextValue)))}
    </div>
  );
}

function renderControl(
  field: RuntimeScalarField | null,
  value: unknown,
  readOnly: boolean | undefined,
  onChange: (nextValue: unknown) => void,
) {
  const commonProps = {
    className: styles.slotControl,
    disabled: Boolean(readOnly),
  };

  if (field?.input === "textarea") {
    return (
      <textarea
        {...commonProps}
        className={`${styles.slotControl} ${styles.slotTextarea}`}
        value={normalizeValue(value)}
        readOnly={readOnly}
        onChange={(event: ChangeEvent<HTMLTextAreaElement>) => {
          if (readOnly) {
            return;
          }
          onChange(event.target.value);
        }}
        rows={5}
      />
    );
  }

  if (field?.input === "select") {
    return (
      <select
        {...commonProps}
        value={normalizeValue(value)}
        onChange={(event: ChangeEvent<HTMLSelectElement>) => {
          if (readOnly) {
            return;
          }
          onChange(event.target.value);
        }}
      >
        <option value="">Selecione</option>
        {field.options.map((option) => (
          <option key={option} value={option}>
            {option}
          </option>
        ))}
      </select>
    );
  }

  if (field?.input === "number") {
    return (
      <input
        {...commonProps}
        type="number"
        value={normalizeNumberValue(value)}
        onChange={(event: ChangeEvent<HTMLInputElement>) => {
          if (readOnly) {
            return;
          }
          const nextValue = event.target.value;
          onChange(nextValue === "" ? "" : Number(nextValue));
        }}
      />
    );
  }

  if (field?.input === "checkbox") {
    return (
      <label className={`${styles.slotControl} ${styles.slotCheckboxRow}`}>
        <input
          type="checkbox"
          checked={Boolean(value)}
          disabled={Boolean(readOnly)}
          onChange={(event) => {
            if (readOnly) {
              return;
            }
            onChange(event.target.checked);
          }}
        />
        <span>{Boolean(value) ? "Sim" : "Nao"}</span>
      </label>
    );
  }

  return (
    <input
      {...commonProps}
      type="text"
      value={normalizeValue(value)}
      onChange={(event: ChangeEvent<HTMLInputElement>) => {
        if (readOnly) {
          return;
        }
        onChange(event.target.value);
      }}
    />
  );
}

function normalizeValue(value: unknown): string {
  if (typeof value === "string") {
    return value;
  }
  if (value === null || value === undefined) {
    return "";
  }
  return String(value);
}

function normalizeNumberValue(value: unknown): string {
  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value);
  }
  if (typeof value === "string" && value.trim()) {
    return value;
  }
  return "";
}

function prettifyPath(path: string): string {
  const lastSegment = path.split(".").pop() ?? path;
  return lastSegment
    .split(/[_-]+/)
    .join(" ")
    .replace(/([a-z])([A-Z])/g, "$1 $2")
    .trim()
    .replace(/^./, (char) => char.toUpperCase());
}
