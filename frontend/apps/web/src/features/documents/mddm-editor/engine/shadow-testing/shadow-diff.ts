import { normalizeDocxXml } from "../golden/golden-helpers";

export type ShadowDiffResult = {
  current_xml_hash: string;
  shadow_xml_hash: string;
  diff_summary: {
    identical: boolean;
    current_length: number;
    shadow_length: number;
    first_divergence_index?: number;
  };
};

async function sha256(input: string): Promise<string> {
  const bytes = new TextEncoder().encode(input);
  const digest = await globalThis.crypto.subtle.digest("SHA-256", bytes);
  return Array.from(new Uint8Array(digest))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

export async function hashNormalizedXml(xml: string): Promise<string> {
  return sha256(normalizeDocxXml(xml));
}

// Synchronous quick-hash for use in computeShadowDiff (avoids async in the
// diff result itself; real SHA-256 is available via hashNormalizedXml when await is possible).
function quickHash(s: string): string {
  let h = 0xcbf29ce4;
  for (let i = 0; i < s.length; i++) {
    h = (h ^ s.charCodeAt(i)) * 0x01000193;
    h >>>= 0;
  }
  return h.toString(16).padStart(8, "0");
}

export function computeShadowDiff(
  currentXml: string,
  shadowXml: string,
): ShadowDiffResult {
  const currentNorm = normalizeDocxXml(currentXml);
  const shadowNorm = normalizeDocxXml(shadowXml);

  const identical = currentNorm === shadowNorm;
  let firstDivergence: number | undefined;
  if (!identical) {
    const min = Math.min(currentNorm.length, shadowNorm.length);
    for (let i = 0; i < min; i++) {
      if (currentNorm[i] !== shadowNorm[i]) {
        firstDivergence = i;
        break;
      }
    }
    firstDivergence ??= min;
  }

  return {
    current_xml_hash: quickHash(currentNorm),
    shadow_xml_hash: quickHash(shadowNorm),
    diff_summary: {
      identical,
      current_length: currentNorm.length,
      shadow_length: shadowNorm.length,
      first_divergence_index: firstDivergence,
    },
  };
}
