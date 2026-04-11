// Feature flag registry. Flags are read once at module load time from a
// window-level config object injected by the backend via the HTML shell.
// Future work (Plan 4): replace with a per-user config endpoint.

import { isInRolloutBucket } from "./feature-flags/rollout";

type FeatureFlags = Readonly<{
  /** Percentage (0..100) of users for whom the new client-side MDDM DOCX path is active. */
  MDDM_NATIVE_EXPORT_ROLLOUT_PCT: number;
  /** Always false at module level — use isMddmNativeExportEnabled(userId) for per-user check. */
  MDDM_NATIVE_EXPORT: boolean;
}>;

function readFlags(): FeatureFlags {
  const injected =
    typeof window !== "undefined"
      ? (
          window as unknown as {
            __METALDOCS_FEATURE_FLAGS?: Partial<{ MDDM_NATIVE_EXPORT_ROLLOUT_PCT: number }>;
          }
        ).__METALDOCS_FEATURE_FLAGS
      : undefined;

  const pct = Number(injected?.MDDM_NATIVE_EXPORT_ROLLOUT_PCT);
  const rolloutPct = Number.isFinite(pct) ? Math.max(0, Math.min(100, pct)) : 0;

  return {
    MDDM_NATIVE_EXPORT_ROLLOUT_PCT: rolloutPct,
    MDDM_NATIVE_EXPORT: false,
  };
}

export const featureFlags: FeatureFlags = readFlags();

/** Returns true when the given userId is inside the canary rollout bucket. */
export function isMddmNativeExportEnabled(userId: string): boolean {
  return isInRolloutBucket(userId, featureFlags.MDDM_NATIVE_EXPORT_ROLLOUT_PCT);
}
