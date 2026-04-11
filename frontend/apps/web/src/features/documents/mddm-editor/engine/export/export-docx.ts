import type { MDDMEnvelope } from "../../adapter";
import { defaultLayoutTokens, type LayoutTokens } from "../layout-ir";
import { canonicalizeAndMigrate } from "../canonicalize-migrate";
import { mddmToDocx } from "../docx-emitter";

const DOCX_MIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document";

export async function exportDocx(
  envelope: MDDMEnvelope,
  tokens?: LayoutTokens,
): Promise<Blob> {
  const canonical = await canonicalizeAndMigrate(envelope);
  const blob = await mddmToDocx(canonical, tokens ?? defaultLayoutTokens);
  // Ensure the returned Blob always carries the correct MIME type.
  if (blob.type === DOCX_MIME) return blob;
  const bytes = await blob.arrayBuffer();
  return new Blob([bytes], { type: DOCX_MIME });
}
