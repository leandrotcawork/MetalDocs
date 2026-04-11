import type { MDDMEnvelope } from "../../adapter";
import type { LayoutTokens } from "../layout-ir";
import { canonicalizeAndMigrate } from "../canonicalize-migrate";
import { mddmToDocx } from "../docx-emitter";

export async function exportDocx(
  envelope: MDDMEnvelope,
  tokens: LayoutTokens,
): Promise<Blob> {
  const canonical = await canonicalizeAndMigrate(envelope);
  return mddmToDocx(canonical, tokens);
}
