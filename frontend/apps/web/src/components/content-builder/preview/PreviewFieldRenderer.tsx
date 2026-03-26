import type { SchemaField } from "../contentSchemaTypes";
import { PreviewTextField } from "./PreviewTextField";
import { PreviewTableField } from "./PreviewTableField";
import { PreviewArrayField } from "./PreviewArrayField";
import { PreviewChecklistField } from "./PreviewChecklistField";

type PreviewFieldRendererProps = {
  field: SchemaField;
  value: unknown;
};

export function PreviewFieldRenderer({ field, value }: PreviewFieldRendererProps) {
  const fieldType = field.type ?? "text";
  const label = field.label ?? field.key;

  if (fieldType === "table") {
    return (
      <PreviewTableField
        label={label}
        rows={Array.isArray(value) ? (value as Record<string, unknown>[]) : []}
        columns={field.columns ?? []}
      />
    );
  }

  if (fieldType === "array") {
    return (
      <PreviewArrayField
        label={label}
        items={Array.isArray(value) ? (value as string[]) : []}
      />
    );
  }

  if (fieldType === "checklist") {
    return <PreviewChecklistField label={label} value={value} />;
  }

  if (fieldType === "select") {
    return <PreviewTextField label={label} value={String(value ?? "")} />;
  }

  if (fieldType === "number") {
    const display = value !== undefined && value !== null && value !== "" ? String(value) : "";
    return <PreviewTextField label={label} value={display} />;
  }

  return <PreviewTextField label={label} value={String(value ?? "")} />;
}
