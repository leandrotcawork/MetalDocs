import type { DocumentTemplateSnapshotItem } from "../../../lib.types";
import { PO_GOVERNED_CANVAS_TEMPLATE, PO_GOVERNED_CANVAS_TEMPLATE_DEFINITION } from "./pilotTemplates";
import type { CanvasTemplateNode, CanvasTemplatePage } from "./templateTypes";

function asRecord(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return {};
  }
  return value as Record<string, unknown>;
}

function isExactKeys(record: Record<string, unknown>, keys: string[]): boolean {
  const actualKeys = Object.keys(record);
  return actualKeys.length === keys.length && keys.every((key) => Object.prototype.hasOwnProperty.call(record, key));
}

function normalizeNodeId(value: unknown): string {
  return typeof value === "string" && value.trim() ? value.trim() : "";
}

function normalizeText(value: unknown): string {
  return typeof value === "string" && value.trim() ? value.trim() : "";
}

function normalizeChildren(value: unknown): CanvasTemplateNode[] | null {
  if (!Array.isArray(value) || value.length === 0) {
    return null;
  }

  const nodes = value.map((child) => normalizeTemplateNode(child));
  if (nodes.some((child) => child === null)) {
    return null;
  }

  return nodes as CanvasTemplateNode[];
}

export function normalizeTemplateNode(value: unknown): CanvasTemplateNode | null {
  const record = asRecord(value);
  const type = typeof record.type === "string" ? record.type : "";

  if (type === "page") {
    if (!isExactKeys(record, ["type", "id", "children"])) {
      return null;
    }
    const id = normalizeNodeId(record.id);
    const children = normalizeChildren(record.children);
    if (!id || !children) {
      return null;
    }
    return { type, id, children };
  }

  if (type === "section-frame") {
    if (!isExactKeys(record, ["type", "id", "title", "children"]) && !isExactKeys(record, ["type", "id", "children"])) {
      return null;
    }
    const id = normalizeNodeId(record.id);
    const title = normalizeText(record.title);
    const children = normalizeChildren(record.children);
    if (!id || !children) {
      return null;
    }
    return title ? { type, id, title, children } : { type, id, children };
  }

  if (type === "label") {
    if (!isExactKeys(record, ["type", "id", "text"])) {
      return null;
    }
    const id = normalizeNodeId(record.id);
    const text = normalizeText(record.text);
    if (!id || !text) {
      return null;
    }
    return { type, id, text };
  }

  if (type === "field-slot" || type === "rich-slot" || type === "table-slot" || type === "repeat-slot") {
    if (!isExactKeys(record, ["type", "id", "path", "fieldKind"])) {
      return null;
    }
    const id = normalizeNodeId(record.id);
    const path = normalizeText(record.path);
    const fieldKind = normalizeText(record.fieldKind);
    if (!id || !path) {
      return null;
    }
    if (type === "field-slot") {
      if (fieldKind !== "scalar") return null;
      return { type, id, path, fieldKind: "scalar" };
    }
    if (type === "rich-slot") {
      if (fieldKind !== "rich") return null;
      return { type, id, path, fieldKind: "rich" };
    }
    if (type === "table-slot") {
      if (fieldKind !== "table") return null;
      return { type, id, path, fieldKind: "table" };
    }
    if (type === "repeat-slot") {
      if (fieldKind !== "repeat") return null;
      return { type, id, path, fieldKind: "repeat" };
    }
  }

  return null;
}

export function normalizeGovernedCanvasTemplate(
  snapshot: DocumentTemplateSnapshotItem | null | undefined,
): CanvasTemplatePage | null {
  if (!snapshot) {
    return null;
  }

  if (
    snapshot.templateKey !== PO_GOVERNED_CANVAS_TEMPLATE.templateKey ||
    snapshot.version !== PO_GOVERNED_CANVAS_TEMPLATE.version ||
    snapshot.profileCode !== PO_GOVERNED_CANVAS_TEMPLATE.profileCode ||
    snapshot.schemaVersion !== PO_GOVERNED_CANVAS_TEMPLATE.schemaVersion
  ) {
    return null;
  }

  const normalized = normalizeTemplateNode(snapshot.definition);
  if (!normalized || normalized.type !== "page") {
    return null;
  }

  if (JSON.stringify(normalized) !== JSON.stringify(PO_GOVERNED_CANVAS_TEMPLATE_DEFINITION)) {
    return null;
  }

  return normalized;
}


