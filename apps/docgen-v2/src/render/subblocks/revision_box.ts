import { SubBlockRenderer, SubBlockContext } from "./registry";

type RevisionEntry = { rev: unknown; date: unknown; description: unknown };

function str(v: unknown): string {
  if (v === null || v === undefined) return "";
  return String(v);
}

function cell(text: string): string {
  return `<w:tc><w:p><w:r><w:t xml:space="preserve">${text}</w:t></w:r></w:p></w:tc>`;
}

function row(a: string, b: string, c: string): string {
  return `<w:tr>${cell(a)}${cell(b)}${cell(c)}</w:tr>`;
}

function isRevisionEntry(v: unknown): v is RevisionEntry {
  return typeof v === "object" && v !== null;
}

export const RevisionBox: SubBlockRenderer = {
  key: "revision_box",
  async render(ctx: SubBlockContext): Promise<string> {
    const raw = ctx.values.revision_history;
    const entries: RevisionEntry[] = Array.isArray(raw) ? raw.filter(isRevisionEntry) : [];

    const header = row("Rev", "Date", "Description");
    const body = entries.length === 0
      ? row("—", "—", "—")
      : entries.map((e) => row(str(e.rev), str(e.date), str(e.description))).join("");

    return `<w:tbl>${header}${body}</w:tbl>`;
  },
};
