import { describe, expect, it } from "vitest";
import {
  loadCurrentRenderer,
  loadPinnedRenderer,
  RendererBundleNotFoundError,
  type LoadedRenderer,
} from "../registry";
import type { RendererPin } from "../../../../../../lib.types";

describe("renderer registry", () => {
  it("loadCurrentRenderer returns a renderer bundle with tokens and mddmToDocx", async () => {
    const renderer = await loadCurrentRenderer();
    expect(renderer.rendererVersion).toMatch(/^\d+\.\d+\.\d+$/);
    expect(typeof renderer.mddmToDocx).toBe("function");
    expect(renderer.tokens.page.widthMm).toBeGreaterThan(0);
    expect(typeof renderer.printStylesheet).toBe("string");
  });

  it("loadPinnedRenderer returns the v1.0.0 bundle for a 1.0.0 pin", async () => {
    const pin: RendererPin = {
      renderer_version: "1.0.0",
      layout_ir_hash: "ignored-for-registry-lookup",
      template_key: "po-mddm-canvas",
      template_version: 1,
    };
    const renderer = await loadPinnedRenderer(pin);
    expect(renderer.rendererVersion).toBe("1.0.0");
  });

  it("loadPinnedRenderer throws RendererBundleNotFoundError for unknown versions", async () => {
    const pin: RendererPin = {
      renderer_version: "9.9.9",
      layout_ir_hash: "h",
      template_key: "k",
      template_version: 1,
    };
    await expect(loadPinnedRenderer(pin)).rejects.toBeInstanceOf(RendererBundleNotFoundError);
  });
});
