import { describe, expect, it } from "vitest";
import {
  canonicalizeAndMigrate,
  CURRENT_MDDM_VERSION,
  MigrationError,
} from "../pipeline";
import type { MDDMEnvelope } from "../../../adapter";

function makeEnvelope(overrides: Partial<MDDMEnvelope> = {}): MDDMEnvelope {
  return {
    mddm_version: CURRENT_MDDM_VERSION,
    template_ref: null,
    blocks: [],
    ...overrides,
  };
}

describe("canonicalizeAndMigrate", () => {
  it("returns the envelope unchanged when already at current version", async () => {
    const envelope = makeEnvelope({
      blocks: [
        { id: "b1", type: "paragraph", props: {}, children: [{ type: "text", text: "hello" }] },
      ],
    });
    const result = await canonicalizeAndMigrate(envelope);
    expect(result.mddm_version).toBe(CURRENT_MDDM_VERSION);
    expect(result.blocks).toHaveLength(1);
  });

  it("sorts object keys for deterministic canonicalization", async () => {
    const envelope = makeEnvelope({
      blocks: [{ zkey: "z", id: "b1", type: "paragraph", props: {}, children: [] } as any],
    });
    const result = await canonicalizeAndMigrate(envelope);
    const firstBlockKeys = Object.keys(result.blocks[0] as Record<string, unknown>);
    const sorted = [...firstBlockKeys].sort();
    expect(firstBlockKeys).toEqual(sorted);
  });

  it("throws MigrationError when version is newer than current", async () => {
    const envelope = makeEnvelope({ mddm_version: CURRENT_MDDM_VERSION + 100 });
    await expect(canonicalizeAndMigrate(envelope)).rejects.toBeInstanceOf(MigrationError);
  });

  it("throws MigrationError when version is missing", async () => {
    const envelope = { template_ref: null, blocks: [] } as unknown as MDDMEnvelope;
    await expect(canonicalizeAndMigrate(envelope)).rejects.toBeInstanceOf(MigrationError);
  });
});
