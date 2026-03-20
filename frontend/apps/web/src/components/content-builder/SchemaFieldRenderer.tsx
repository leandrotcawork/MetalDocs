import type { SchemaField } from "./contentSchemaTypes";
import { ArrayField } from "./widgets/ArrayField";
import { ChecklistField } from "./widgets/ChecklistField";
import { SelectField } from "./widgets/SelectField";
import { TableField } from "./widgets/TableField";
import { TextAreaField } from "./widgets/TextAreaField";
import { TextField } from "./widgets/TextField";

type SchemaFieldRendererProps = {
  field: SchemaField;
  value: unknown;
  onChange: (next: unknown) => void;
};

export function SchemaFieldRenderer({ field, value, onChange }: SchemaFieldRendererProps) {
  const fieldType = field.type ?? "text";
  const label = field.label ?? field.key;
  const required = Boolean(field.required);

  if (fieldType === "textarea") {
    return (
      <TextAreaField
        label={label}
        required={required}
        value={(value as string) ?? ""}
        onChange={onChange}
      />
    );
  }

  if (fieldType === "select") {
    return (
      <SelectField
        label={label}
        required={required}
        value={(value as string) ?? ""}
        options={field.options ?? []}
        onChange={onChange}
      />
    );
  }

  if (fieldType === "number") {
    const numericValue = typeof value === "number" ? String(value) : (value as string | undefined) ?? "";
    return (
      <TextField
        label={label}
        required={required}
        value={numericValue}
        type="number"
        onChange={(next) => onChange(next === "" ? "" : Number(next))}
      />
    );
  }

  if (fieldType === "array") {
    return (
      <ArrayField
        label={label}
        required={required}
        items={Array.isArray(value) ? (value as string[]) : []}
        onChange={onChange}
      />
    );
  }

  if (fieldType === "checklist") {
    return <ChecklistField label={label} required={required} value={value} onChange={onChange} />;
  }

  if (fieldType === "table") {
    return (
      <TableField
        label={label}
        required={required}
        rows={Array.isArray(value) ? (value as Record<string, unknown>[]) : []}
        columns={field.columns ?? []}
        onChange={onChange}
      />
    );
  }

  return (
    <TextField
      label={label}
      required={required}
      value={(value as string) ?? ""}
      onChange={onChange}
    />
  );
}
