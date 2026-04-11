import { describe, expect, it } from "vitest";
import { computeLayoutIRHash, serializeLayoutIRForHash } from "../compute-ir-hash";
import { defaultLayoutTokens, defaultComponentRules } from "../../layout-ir";

describe("computeLayoutIRHash", () => {
  it("produces a stable hex SHA-256 digest", async () => {
    const hash = await computeLayoutIRHash({ tokens: defaultLayoutTokens, components: defaultComponentRules });
    expect(hash).toMatch(/^[0-9a-f]{64}$/);
  });

  it("produces the same hash for the same input", async () => {
    const a = await computeLayoutIRHash({ tokens: defaultLayoutTokens, components: defaultComponentRules });
    const b = await computeLayoutIRHash({ tokens: defaultLayoutTokens, components: defaultComponentRules });
    expect(a).toBe(b);
  });

  it("produces a different hash when a token changes", async () => {
    const a = await computeLayoutIRHash({ tokens: defaultLayoutTokens, components: defaultComponentRules });
    const changed = {
      ...defaultLayoutTokens,
      page: { ...defaultLayoutTokens.page, widthMm: 999 },
    };
    const b = await computeLayoutIRHash({ tokens: changed, components: defaultComponentRules });
    expect(a).not.toBe(b);
  });

  it("serializeLayoutIRForHash produces deterministic key order", () => {
    const serialized1 = serializeLayoutIRForHash({ tokens: defaultLayoutTokens, components: defaultComponentRules });
    const serialized2 = serializeLayoutIRForHash({ tokens: defaultLayoutTokens, components: defaultComponentRules });
    expect(serialized1).toBe(serialized2);
  });
});
