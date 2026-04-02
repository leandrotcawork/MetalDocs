const EMPTY_DOC = {
  type: "doc",
  content: [{ type: "paragraph" }],
} as const;

export const METALDOCS_RICH_ENVELOPE_FORMAT = "metaldocs.rich.tiptap";
export const METALDOCS_RICH_ENVELOPE_VERSION = 1;

export function toEnvelope(editorJSON: Record<string, unknown>) {
  return {
    format: METALDOCS_RICH_ENVELOPE_FORMAT,
    version: METALDOCS_RICH_ENVELOPE_VERSION,
    content: editorJSON,
  };
}

export function fromEnvelope(value: unknown): Record<string, unknown> {
  if (isRichEnvelope(value) && isRecord(value.content)) {
    return normalizeDoc(value.content);
  }

  if (typeof value === "string" && value.trim()) {
    return {
      type: "doc",
      content: [
        {
          type: "paragraph",
          content: [{ type: "text", text: value }],
        },
      ],
    };
  }

  return normalizeDoc(EMPTY_DOC);
}

function isRichEnvelope(value: unknown): value is { format: string; version: number; content: unknown } {
  if (!isRecord(value)) {
    return false;
  }
  return value.format === METALDOCS_RICH_ENVELOPE_FORMAT && value.version === METALDOCS_RICH_ENVELOPE_VERSION;
}

function normalizeDoc(value: Record<string, unknown>): Record<string, unknown> {
  const type = typeof value.type === "string" ? value.type : "";
  const content = Array.isArray(value.content) ? value.content : [];

  if (type !== "doc" || content.length === 0) {
    return {
      type: "doc",
      content: [{ type: "paragraph" }],
    };
  }

  return {
    type: "doc",
    content,
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}
