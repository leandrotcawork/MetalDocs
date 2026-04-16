/**
 * Strip a leading `#` and uppercase the result so the string can be used
 * directly as a docx OOXML color value (e.g. "DFC8C8").
 */
export function hexToFill(hex: string): string {
  return hex.replace(/^#/, "").toUpperCase();
}
