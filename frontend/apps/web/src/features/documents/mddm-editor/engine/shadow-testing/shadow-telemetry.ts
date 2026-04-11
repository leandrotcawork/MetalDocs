export type ShadowDiffPayload = {
  document_id: string;
  version_number: number;
  user_id_hash: string;
  current_xml_hash: string;
  shadow_xml_hash: string;
  diff_summary: Record<string, unknown>;
  current_duration_ms: number;
  shadow_duration_ms: number;
  shadow_error?: string;
};

/**
 * Fire-and-forget POST to the shadow-diff telemetry endpoint.
 * Intentionally swallows errors — the user-visible export must not
 * be affected by telemetry failures.
 */
export async function postShadowDiff(payload: ShadowDiffPayload): Promise<void> {
  try {
    await fetch("/api/v1/telemetry/mddm-shadow-diff", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
  } catch {
    // Swallow: telemetry is best-effort.
  }
}
