export function normalizeDocumentProfileCode(value?: string): string {
  return (value ?? "").trim().toLowerCase();
}
