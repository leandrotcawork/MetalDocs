import { describe, it, expect, beforeEach, vi } from 'vitest';

describe('DOCX_V2_ENABLED flag', () => {
  beforeEach(() => {
    vi.resetModules();
    (window as unknown as { __METALDOCS_FEATURE_FLAGS?: Record<string, unknown> })
      .__METALDOCS_FEATURE_FLAGS = undefined;
  });

  it('defaults to false when no source provides it', async () => {
    const { featureFlags, isDocxV2Enabled } = await import('../featureFlags');
    expect(featureFlags.DOCX_V2_ENABLED).toBe(false);
    expect(isDocxV2Enabled()).toBe(false);
  });

  it('reads true from window injection', async () => {
    (window as unknown as { __METALDOCS_FEATURE_FLAGS?: Record<string, unknown> })
      .__METALDOCS_FEATURE_FLAGS = { DOCX_V2_ENABLED: true };
    const { featureFlags, isDocxV2Enabled } = await import('../featureFlags');
    expect(featureFlags.DOCX_V2_ENABLED).toBe(true);
    expect(isDocxV2Enabled()).toBe(true);
  });

  it('treats non-boolean truthy as false (strict)', async () => {
    (window as unknown as { __METALDOCS_FEATURE_FLAGS?: Record<string, unknown> })
      .__METALDOCS_FEATURE_FLAGS = { DOCX_V2_ENABLED: 'true' };
    const { featureFlags } = await import('../featureFlags');
    expect(featureFlags.DOCX_V2_ENABLED).toBe(false);
  });

  it('initFeatureFlags patches from /api/v1/feature-flags', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ DOCX_V2_ENABLED: true }), { status: 200 })
    );
    const mod = await import('../featureFlags');
    await mod.initFeatureFlags();
    expect(mod.featureFlags.DOCX_V2_ENABLED).toBe(true);
    fetchSpy.mockRestore();
  });
});
