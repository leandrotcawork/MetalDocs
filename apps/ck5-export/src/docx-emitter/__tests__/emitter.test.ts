import { describe, expect, it } from "vitest";
import { mddmToDocx, MissingEmitterError } from "../emitter";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMEnvelope } from "../../shared/adapter";

describe("mddmToDocx", () => {
  it("returns a Blob for a paragraph-only envelope", async () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "p1",
          type: "paragraph",
          props: {},
          children: [{ type: "text", text: "Hello" }],
        },
      ],
    };
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    expect(blob).toBeInstanceOf(Blob);
    expect(blob.size).toBeGreaterThan(0);
    expect(blob.type).toBe(
      "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    );
  });

  it("returns a Blob for a section + field envelope", async () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        { id: "s1", type: "section", props: { title: "1. Procedimento" }, children: [] },
        {
          id: "f1",
          type: "field",
          props: { label: "ResponsÃ¡vel" },
          children: [{ type: "text", text: "JoÃ£o" }],
        },
      ],
    };
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    expect(blob.size).toBeGreaterThan(0);
  });

  it("throws MissingEmitterError for unknown block types", async () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [{ id: "x", type: "unknownXYZ", props: {}, children: [] }],
    };
    await expect(mddmToDocx(envelope, defaultLayoutTokens)).rejects.toBeInstanceOf(
      MissingEmitterError,
    );
  });
});

