import type { DocumentListItem, DocumentProfileItem } from "../../lib.types";

type DocumentIdentity = Pick<DocumentListItem, "documentId" | "title" | "documentProfile" | "documentCode">;

function normalizeToken(value?: string): string {
  return (value ?? "").trim();
}

function profileShortCode(profileCode: string, profiles?: DocumentProfileItem[]): string {
  const code = normalizeToken(profileCode).toLowerCase();
  const profile = profiles?.find((item) => item.code.trim().toLowerCase() === code);

  const alias = normalizeToken(profile?.alias);
  if (alias && alias.length <= 3) return alias.toUpperCase();

  if (code && code.length <= 3) return code.toUpperCase();

  if (alias) return alias.split(/[^A-Za-z0-9]+/).filter(Boolean)[0]?.slice(0, 3).toUpperCase() ?? code.toUpperCase();
  return code.toUpperCase() || "DOC";
}

function extractSequenceFromId(documentId: string, prefix: string): string | null {
  const id = normalizeToken(documentId);
  if (!id) return null;

  // Common patterns:
  // - PO-441, IT_12, RG 7
  // - po-441-<hash> (we only care about the first number)
  const pref = normalizeToken(prefix);
  if (pref) {
    const re = new RegExp(String.raw`^${pref}[-_ ]?(\\d{1,6})\\b`, "i");
    const match = id.match(re);
    if (match?.[1]) return match[1];
  }

  const generic = id.match(/\b(\d{1,6})\b/);
  return generic?.[1] ?? null;
}

function alreadyStandardized(title: string): boolean {
  return /^[A-Z0-9]{2,3}-\d{1,6}-/.test(title.trim());
}

export function formatDocumentDisplayName(doc: DocumentIdentity, profiles?: DocumentProfileItem[]): string {
  const title = normalizeToken(doc.title) || "Documento";
  if (alreadyStandardized(title)) return title;

  const canonicalCode = normalizeToken(doc.documentCode);
  if (canonicalCode) return `${canonicalCode} ${title}`;

  const prefix = profileShortCode(doc.documentProfile, profiles);
  const seq = extractSequenceFromId(doc.documentId, prefix);
  if (!seq) return `${prefix} ${title}`;
  return `${prefix}-${seq} ${title}`;
}
