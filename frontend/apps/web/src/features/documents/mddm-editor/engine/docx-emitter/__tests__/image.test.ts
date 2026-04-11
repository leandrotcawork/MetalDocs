import { describe, expect, it } from "vitest";
import { Paragraph, ImageRun } from "docx";
import { emitImage, MissingAssetError } from "../emitters/image";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import type { ResolvedAsset } from "../../asset-resolver";

const PNG_BYTES = new Uint8Array([0x89, 0x50, 0x4e, 0x47]);

function makeAsset(): ResolvedAsset {
  return { bytes: PNG_BYTES, mimeType: "image/png", sizeBytes: PNG_BYTES.byteLength };
}

describe("emitImage", () => {
  it("emits a Paragraph containing an ImageRun for a resolved image (src prop)", () => {
    const block: MDDMBlock = {
      id: "i1",
      type: "image",
      props: { src: "/api/images/aaa", widthMm: 80 },
      children: [],
    };
    const map = new Map<string, ResolvedAsset>([["/api/images/aaa", makeAsset()]]);
    const out = emitImage(block, defaultLayoutTokens, map);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
    expect((out[0] as any).options.children[0]).toBeInstanceOf(ImageRun);
  });

  it("throws MissingAssetError when image src is not in the asset map", () => {
    const block: MDDMBlock = {
      id: "i2",
      type: "image",
      props: { src: "/api/images/missing" },
      children: [],
    };
    expect(() => emitImage(block, defaultLayoutTokens, new Map())).toThrow(MissingAssetError);
  });

  it("returns an empty Paragraph when block has no src prop", () => {
    const block: MDDMBlock = {
      id: "i3",
      type: "image",
      props: {},
      children: [],
    };
    const out = emitImage(block, defaultLayoutTokens, new Map());
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
  });
});
