import { describe, expect, it } from "vitest";
import { computeLayoutIRHash } from "../compute-ir-hash";
import { defaultLayoutTokens, defaultComponentRules } from "../../layout-ir";
import { RECORDED_IR_HASH, RECORDED_RENDERER_VERSION } from "../recorded-hash";

describe("Layout IR drift gate", () => {
  it("current IR hash matches the recorded hash", async () => {
    const current = await computeLayoutIRHash({
      tokens: defaultLayoutTokens,
      components: defaultComponentRules,
    });

    if ((RECORDED_IR_HASH as string) === "PLACEHOLDER_REGENERATE_VIA_DRIFT_GATE") {
      throw new Error(
        `RECORDED_IR_HASH is a placeholder. Edit recorded-hash.ts and set:\n` +
        `  export const RECORDED_IR_HASH = "${current}";\n` +
        `then commit. This records the current engine as the baseline for future drift detection.`
      );
    }

    expect(current).toBe(RECORDED_IR_HASH);
  });

  it("RECORDED_RENDERER_VERSION is populated", () => {
    expect(RECORDED_RENDERER_VERSION).toMatch(/^\d+\.\d+\.\d+$/);
  });
});
