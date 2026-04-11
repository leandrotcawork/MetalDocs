import { describe, expect, it, vi } from "vitest";
import { exportDocx } from "../export-docx";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMEnvelope } from "../../../adapter";
import * as pipeline from "../../canonicalize-migrate/pipeline";
import * as emitter from "../../docx-emitter/emitter";

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
