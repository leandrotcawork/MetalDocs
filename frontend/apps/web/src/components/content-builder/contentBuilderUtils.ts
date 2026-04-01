import type { SchemaField } from "./contentSchemaTypes";

export function sectionAnchorId(sectionKey: string) {
  return `content-section-${sectionKey}`;
}

export function hasAnyValue(field: SchemaField, value: unknown): boolean {
  if (field.type === "table") {
    return Array.isArray(value) && value.length > 0;
  }
  if (field.type === "repeat") {
    return hasRepeatValue(field.itemFields ?? [], value);
  }
  if (field.type === "array") {
    return Array.isArray(value) && value.some((item) => String(item ?? "").trim() !== "");
  }
  if (field.type === "checklist") {
    return Array.isArray(value) && value.some((item) => typeof item === "string" ? item.trim() !== "" : Boolean((item as { label?: string }).label));
  }
  if (field.type === "number") {
    return value !== null && value !== undefined && value !== "";
  }
  return String(value ?? "").trim() !== "";
}

export function isFieldComplete(field: SchemaField, value: unknown) {
  if (field.type === "table") {
    return Array.isArray(value) && value.length > 0;
  }
  if (field.type === "repeat") {
    return hasRepeatValue(field.itemFields ?? [], value);
  }
  if (field.type === "array") {
    return Array.isArray(value) && value.some((item) => String(item ?? "").trim() !== "");
  }
  if (field.type === "checklist") {
    return Array.isArray(value) && value.some((item) => typeof item === "string" ? item.trim() !== "" : Boolean((item as { label?: string }).label));
  }
  if (field.type === "number") {
    return value !== null && value !== undefined && value !== "";
  }
  return String(value ?? "").trim() !== "";
}

export function sectionProgress(fields: SchemaField[], sectionValue: Record<string, unknown>) {
  const total = fields.length;
  if (total === 0) {
    return { progressDots: 0, progressLabel: "0%" };
  }
  const completed = fields.reduce((acc, field) => {
    const value = sectionValue[field.key];
    return acc + (isFieldComplete(field, value) ? 1 : 0);
  }, 0);
  const ratio = Math.round((completed / total) * 100);
  return {
    progressDots: ratio === 0 ? 0 : ratio < 50 ? 1 : ratio < 90 ? 2 : 3,
    progressLabel: `${ratio}%`,
  };
}

function hasRepeatValue(itemFields: SchemaField[], value: unknown): boolean {
  if (!Array.isArray(value) || value.length === 0) {
    return false;
  }
  if (itemFields.length === 0) {
    return true;
  }
  return value.some((item) => {
    if (!item || typeof item !== "object" || Array.isArray(item)) {
      return String(item ?? "").trim() !== "";
    }
    const record = item as Record<string, unknown>;
    return itemFields.some((field) => hasAnyValue(field, record[field.key]));
  });
}
