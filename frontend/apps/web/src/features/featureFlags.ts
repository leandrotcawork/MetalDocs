// Feature flag registry. Flags are read once at module load time from a
// window-level config object injected by the backend via the HTML shell.
// Future work (Plan 4): replace with a per-user config endpoint.

type FeatureFlags = Readonly<{
  MDDM_NATIVE_EXPORT: boolean;
}>;

function readFlags(): FeatureFlags {
  const injected = typeof window !== "undefined"
    ? (window as unknown as { __METALDOCS_FEATURE_FLAGS?: Partial<FeatureFlags> }).__METALDOCS_FEATURE_FLAGS
    : undefined;

  return {
    MDDM_NATIVE_EXPORT: injected?.MDDM_NATIVE_EXPORT === true,
  };
}

export const featureFlags: FeatureFlags = readFlags();
