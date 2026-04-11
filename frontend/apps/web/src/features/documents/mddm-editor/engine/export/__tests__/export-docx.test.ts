import { describe, expect, it, vi } from "vitest";
import { exportDocx } from "../export-docx";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMEnvelope } from "../../../adapter";
import * as pipeline from "../../canonicalize-migrate/pipeline";
import * as emitter from "../../docx-emitter/emitter";
import { AssetResolver } from "../../asset-resolver";

const minimalEnvelope: MDDMEnvelope = {
  mddm_version: 1,
  template_ref: null,
  blocks: [
    {
      id: "p1",
      type: "paragraph",
      props: {},
      children: [{ type: "text", text: "Hello world" }],
    },
  ],
};

describe("exportDocx", () => {
  it("generates a DOCX Blob for a simple envelope", async () => {
    const blob = await exportDocx(minimalEnvelope, defaultLayoutTokens);
    expect(blob).toBeInstanceOf(Blob);
    expect(blob.size).toBeGreaterThan(100);
    expect(blob.type).toBe("application/vnd.openxmlformats-officedocument.wordprocessingml.document");
  });

  it("calls canonicalizeAndMigrate before mddmToDocx", async () => {
    const callOrder: string[] = [];
    const canonicalizeSpy = vi.spyOn(pipeline, "canonicalizeAndMigrate").mockImplementation(async (e) => {
      callOrder.push("canonicalize");
      return e;
    });
    const emitSpy = vi.spyOn(emitter, "mddmToDocx").mockResolvedValue(
      new Blob(["x"], { type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document" }),
    );

    await exportDocx(minimalEnvelope);
    expect(callOrder[0]).toBe("canonicalize");
    expect(emitSpy).toHaveBeenCalled();

    canonicalizeSpy.mockRestore();
    emitSpy.mockRestore();
  });

  it("uses defaultLayoutTokens when tokens are omitted", async () => {
    const blob = await exportDocx(minimalEnvelope);
    expect(blob).toBeInstanceOf(Blob);
    expect(blob.size).toBeGreaterThan(0);
  });
});

describe("exportDocx asset wiring", () => {
  it("calls the asset resolver for each unique image URL", async () => {
    const PNG = new Uint8Array([0x89, 0x50, 0x4e, 0x47]);
    const calls: string[] = [];

    const fakeResolver = {
      async resolveAsset(url: string) {
        calls.push(url);
        return { bytes: PNG, mimeType: "image/png" as const, sizeBytes: PNG.byteLength };
      },
    } as unknown as AssetResolver;

    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        { id: "i1", type: "image", props: { src: "/api/images/aaa" }, children: [] },
        { id: "i2", type: "image", props: { src: "/api/images/bbb" }, children: [] },
      ],
    };

    const blob = await exportDocx(envelope, defaultLayoutTokens, { assetResolver: fakeResolver });
    expect(blob).toBeInstanceOf(Blob);
    expect(calls).toEqual(["/api/images/aaa", "/api/images/bbb"]);
  });
});
