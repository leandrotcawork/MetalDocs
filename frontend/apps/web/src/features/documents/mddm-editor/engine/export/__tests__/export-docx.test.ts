import { describe, expect, it } from "vitest";
import { exportDocx } from "../export-docx";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMEnvelope } from "../../../adapter";

describe("exportDocx", () => {
  it("generates a DOCX Blob for a simple envelope", async () => {
    const envelope: MDDMEnvelope = {
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

    const blob = await exportDocx(envelope, defaultLayoutTokens);
    expect(blob).toBeInstanceOf(Blob);
    expect(blob.size).toBeGreaterThan(100);
    expect(blob.type).toBe("application/vnd.openxmlformats-officedocument.wordprocessingml.document");
  });

  it("runs canonicalize+migrate before emitting", async () => {
    const envelope = {
      template_ref: null,
      mddm_version: 1,
      blocks: [
        {
          type: "paragraph",
          id: "p1",
          props: {},
          children: [{ type: "text", text: "x" }],
        },
      ],
    } as unknown as MDDMEnvelope;

    const blob = await exportDocx(envelope, defaultLayoutTokens);
    expect(blob.size).toBeGreaterThan(0);
  });
});
