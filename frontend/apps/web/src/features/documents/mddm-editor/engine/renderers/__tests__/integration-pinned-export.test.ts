import { describe, expect, it } from "vitest";
import { exportDocx } from "../../export";
import type { MDDMEnvelope } from "../../../adapter";
import type { RendererPin } from "../../../../../../lib.types";

describe("Pinned export integration", () => {
  const envelope: MDDMEnvelope = {
    mddm_version: 1,
    template_ref: null,
    blocks: [
      {
        id: "s",
        type: "section",
        props: { title: "Test", color: "red" },
        children: [],
      },
      {
        id: "p",
        type: "paragraph",
        props: {},
        children: [{ type: "text", text: "hello" }],
      },
    ],
  };

  it("released document with v1.0.0 pin produces a valid DOCX", async () => {
    const pin: RendererPin = {
      renderer_version: "1.0.0",
      layout_ir_hash: "placeholder",
      template_key: "po-mddm-canvas",
      template_version: 1,
    };
    const blob = await exportDocx(envelope, { rendererPin: pin });
    expect(blob).toBeInstanceOf(Blob);
    expect(blob.size).toBeGreaterThan(500);
    expect(blob.type).toBe("application/vnd.openxmlformats-officedocument.wordprocessingml.document");
  });

  it("draft document without pin produces a valid DOCX via current renderer", async () => {
    const blob = await exportDocx(envelope, { rendererPin: null });
    expect(blob).toBeInstanceOf(Blob);
    expect(blob.size).toBeGreaterThan(500);
  });

  it("unknown renderer_version rejects cleanly", async () => {
    const pin: RendererPin = {
      renderer_version: "9.9.9",
      layout_ir_hash: "h",
      template_key: "k",
      template_version: 1,
    };
    await expect(exportDocx(envelope, { rendererPin: pin })).rejects.toThrow(/renderer bundle/i);
  });
});
