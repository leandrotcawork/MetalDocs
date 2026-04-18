// Feature flag registry. Flags are read once at module load time from a
// window-level config object injected by the backend via the HTML shell.
// When the window injection is absent, initFeatureFlags() fetches from
// GET /api/v1/feature-flags and patches the exported featureFlags object.

import { isInRolloutBucket } from "./feature-flags/rollout";

type FeatureFlags = {
  /** Percentage (0..100) of users for whom the new client-side MDDM DOCX path is active. */
  MDDM_NATIVE_EXPORT_ROLLOUT_PCT: number;
  /** Always false at module level — use isMddmNativeExportEnabled(userId) for per-user check. */
  MDDM_NATIVE_EXPORT: boolean;
  /** docx-editor v2 platform gate. Strict boolean; default false. */
  DOCX_V2_ENABLED: boolean;
};

function readWindowFlags():
  | Partial<{ MDDM_NATIVE_EXPORT_ROLLOUT_PCT: number; DOCX_V2_ENABLED: boolean }>
  | undefined {
  if (typeof window === "undefined") return undefined;
  return (
    window as unknown as {
      __METALDOCS_FEATURE_FLAGS?: Partial<{
        MDDM_NATIVE_EXPORT_ROLLOUT_PCT: number;
        DOCX_V2_ENABLED: boolean;
      }>;
    }
  ).__METALDOCS_FEATURE_FLAGS;
}

function clampPct(raw: unknown): number {
  const n = Number(raw);
  return Number.isFinite(n) ? Math.max(0, Math.min(100, n)) : 0;
}

function strictBool(raw: unknown): boolean {
  return raw === true;
}

function readFlags(): FeatureFlags {
  const injected = readWindowFlags();
  return {
    MDDM_NATIVE_EXPORT_ROLLOUT_PCT: clampPct(injected?.MDDM_NATIVE_EXPORT_ROLLOUT_PCT),
    MDDM_NATIVE_EXPORT: false,
    DOCX_V2_ENABLED: strictBool(injected?.DOCX_V2_ENABLED),
  };
}

export const featureFlags: FeatureFlags = readFlags();

/**
 * Fetches feature flags from GET /api/v1/feature-flags and patches the
 * exported featureFlags object. Call once at app init (e.g. in main.tsx)
 * before render. Safe to call even when window injection is present — the
 * endpoint value overrides it.
 */
export async function initFeatureFlags(): Promise<void> {
  try {
    const res = await fetch("/api/v1/feature-flags");
    if (!res.ok) return;
    const data = (await res.json()) as Partial<{
      MDDM_NATIVE_EXPORT_ROLLOUT_PCT: number;
      DOCX_V2_ENABLED: boolean;
    }>;
    featureFlags.MDDM_NATIVE_EXPORT_ROLLOUT_PCT = clampPct(data.MDDM_NATIVE_EXPORT_ROLLOUT_PCT);
    featureFlags.DOCX_V2_ENABLED = strictBool(data.DOCX_V2_ENABLED);
  } catch {
    // Network error or non-JSON body — keep defaults
  }
}

/** Returns true when the given userId is inside the canary rollout bucket. */
export function isMddmNativeExportEnabled(userId: string): boolean {
  return isInRolloutBucket(userId, featureFlags.MDDM_NATIVE_EXPORT_ROLLOUT_PCT);
}

/** True iff the docx-editor v2 platform is active for this session. */
export function isDocxV2Enabled(): boolean {
  return featureFlags.DOCX_V2_ENABLED;
}
