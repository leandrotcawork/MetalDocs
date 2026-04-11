import type { LayoutTokens, ComponentRules } from "../layout-ir";

export type LayoutIRSnapshot = {
  tokens: LayoutTokens;
  components: ComponentRules;
};

function stableStringify(value: unknown): string {
  if (value === null || typeof value !== "object") {
    return JSON.stringify(value);
  }
  if (Array.isArray(value)) {
    return `[${value.map(stableStringify).join(",")}]`;
  }
  const entries = Object.entries(value as Record<string, unknown>).sort(
    ([a], [b]) => (a < b ? -1 : a > b ? 1 : 0),
  );
  return `{${entries.map(([k, v]) => `${JSON.stringify(k)}:${stableStringify(v)}`).join(",")}}`;
}

export function serializeLayoutIRForHash(snapshot: LayoutIRSnapshot): string {
  return stableStringify(snapshot);
}

export async function computeLayoutIRHash(snapshot: LayoutIRSnapshot): Promise<string> {
  const serialized = serializeLayoutIRForHash(snapshot);
  const encoder = new TextEncoder();
  const data = encoder.encode(serialized);

  const subtle = (globalThis.crypto && globalThis.crypto.subtle) as SubtleCrypto | undefined;
  if (subtle) {
    const digest = await subtle.digest("SHA-256", data);
    return Array.from(new Uint8Array(digest))
      .map((b) => b.toString(16).padStart(2, "0"))
      .join("");
  }
  throw new Error("Web Crypto not available for layout IR hashing");
}
