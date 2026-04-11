import { describe, expect, it } from "vitest";
import { collectImageUrls } from "../asset-collector";
import type { MDDMEnvelope } from "../../../adapter";

describe("collectImageUrls", () => {
  it("returns an empty array when there are no image blocks", () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        { id: "p", type: "paragraph", props: {}, children: [{ type: "text", text: "x" }] },
      ],
    };
    expect(collectImageUrls(envelope)).toEqual([]);
  });

  it("returns image URLs from top-level image blocks (reads block.props.src)", () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        { id: "i1", type: "image", props: { src: "/api/images/aaa" }, children: [] },
        { id: "i2", type: "image", props: { src: "/api/images/bbb" }, children: [] },
      ],
    };
    expect(collectImageUrls(envelope)).toEqual(["/api/images/aaa", "/api/images/bbb"]);
  });

  it("walks nested children for images inside repeatables and sections", () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "s",
          type: "section",
          props: { title: "S" },
          children: [
            {
              id: "r",
              type: "repeatable",
              props: { label: "L" },
              children: [
                {
                  id: "ri",
                  type: "repeatableItem",
                  props: {},
                  children: [
                    { id: "img", type: "image", props: { src: "/api/images/nested" }, children: [] },
                  ],
                },
              ],
            },
          ],
        },
      ],
    };
    expect(collectImageUrls(envelope)).toEqual(["/api/images/nested"]);
  });

  it("deduplicates URLs", () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        { id: "i1", type: "image", props: { src: "/api/images/aaa" }, children: [] },
        { id: "i2", type: "image", props: { src: "/api/images/aaa" }, children: [] },
      ],
    };
    expect(collectImageUrls(envelope)).toEqual(["/api/images/aaa"]);
  });

  it("ignores image blocks without a src prop", () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        { id: "i1", type: "image", props: {}, children: [] },
        { id: "i2", type: "image", props: { src: "" }, children: [] },
      ],
    };
    expect(collectImageUrls(envelope)).toEqual([]);
  });
});
