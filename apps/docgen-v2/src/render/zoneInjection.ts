export async function injectZones(
  docx: Uint8Array,
  _zones: Record<string, string>,
): Promise<Uint8Array> {
  return docx;
}
