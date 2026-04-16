import { describe, expect, it } from "vitest";
import { rewriteImgSrcToDataUri } from "../inline-asset-rewriter";
import type { ResolvedAsset } from "../asset-resolver";

function makeAsset(byte: number): ResolvedAsset {
  return {
    bytes: new Uint8Array([byte, byte, byte, byte]),
    mimeType: "image/png",
    sizeBytes: 4,
  };
}

describe("rewriteImgSrcToDataUri", () => {
  it("rewrites a single img src to a data: URI", () => {
    const html = `<p><img src="/api/images/aaa" alt="A" /></p>`;
    const map = new Map<string, ResolvedAsset>([
      ["/api/images/aaa", makeAsset(0x01)],
    ]);
    const out = rewriteImgSrcToDataUri(html, map);
    // 0x01 0x01 0x01 0x01 encodes to "AQEBAQ==" in base64
    expect(out).toContain("data:image/png;base64,AQEBAQ==");
    expect(out).not.toContain("/api/images/aaa");
    expect(out).toContain('alt="A"');
  });

  it("rewrites multiple img tags with different URLs", () => {
    const html = `<img src="/api/images/aaa"/><img src="/api/images/bbb"/>`;
    const map = new Map<string, ResolvedAsset>([
      ["/api/images/aaa", makeAsset(0x10)],
      ["/api/images/bbb", makeAsset(0x20)],
    ]);
    const out = rewriteImgSrcToDataUri(html, map);
    const matches = out.match(/data:image\/png;base64,/g);
    expect(matches).toHaveLength(2);
  });

  it("leaves img tags whose src is not in the map untouched", () => {
    const html = `<img src="/api/images/missing"/>`;
    const map = new Map<string, ResolvedAsset>();
    const out = rewriteImgSrcToDataUri(html, map);
    expect(out).toContain("/api/images/missing");
  });

  it("handles single-quoted src attributes", () => {
    const html = `<img src='/api/images/aaa'/>`;
    const map = new Map<string, ResolvedAsset>([
      ["/api/images/aaa", makeAsset(0x01)],
    ]);
    const out = rewriteImgSrcToDataUri(html, map);
    expect(out).toContain("data:image/png;base64,");
  });
});
