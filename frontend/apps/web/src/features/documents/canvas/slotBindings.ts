import type { RuntimeDocumentSchema, RuntimeField, RuntimeSection } from "../runtime/schemaRuntimeTypes";

export type CanvasSlotBinding = {
  section: RuntimeSection;
  field: RuntimeField | null;
  sectionKey: string;
  fieldKey: string;
  label: string;
  description: string;
  required: boolean;
};

export function resolveCanvasSlotBinding(schema: RuntimeDocumentSchema | null, path: string): CanvasSlotBinding | null {
  if (!schema) {
    return null;
  }

  const segments = path
    .split(".")
    .map((segment) => segment.trim())
    .filter(Boolean);

  if (segments.length < 2) {
    return null;
  }

  const sectionKey = segments[0];
  const fieldKey = segments.slice(1).join(".");
  const section = schema.sections.find((item) => item.key === sectionKey);

  if (!section) {
    return null;
  }

  const field = section.fields.find((item) => item.key === fieldKey) ?? null;
  return {
    section,
    field,
    sectionKey,
    fieldKey,
    label: field?.label?.trim() || prettifyKey(fieldKey),
    description: field?.description?.trim() || "",
    required: Boolean(field?.required),
  };
}

function prettifyKey(value: string): string {
  const cleaned = value
    .split(/[_-]+/)
    .join(" ")
    .replace(/([a-z])([A-Z])/g, "$1 $2")
    .trim();

  if (!cleaned) {
    return value;
  }

  return cleaned.charAt(0).toUpperCase() + cleaned.slice(1);
}
