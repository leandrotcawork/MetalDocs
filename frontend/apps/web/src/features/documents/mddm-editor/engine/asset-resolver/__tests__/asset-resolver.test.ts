import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AssetResolver, AssetResolverError } from "../asset-resolver";

// Minimal PNG (1x1 red pixel). Starts with PNG magic bytes 89 50 4E 47.
const TINY_PNG = new Uint8Array([
  0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
  0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
  0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
  0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
  0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
  0x54, 0x08, 0x99, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
  0x00, 0x00, 0x03, 0x00, 0x01, 0x5a, 0x4d, 0x7f,
  0x5c, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
  0x44, 0xae, 0x42, 0x60, 0x82,
]);

const VALID_URL = "/api/images/00000000-0000-4000-8000-000000000001";

function mockFetchOnce(response: Response): void {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(response));
}

describe("AssetResolver", () => {
  let resolver: AssetResolver;

  beforeEach(() => {
    resolver = new AssetResolver();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("rejects URLs that fail the allowlist before fetching", async () => {
    const fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);

    await expect(resolver.resolveAsset("https://evil.example/pwn.png"))
      .rejects.toBeInstanceOf(AssetResolverError);
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("rejects assets exceeding maxImageSizeBytes", async () => {
    const huge = new Uint8Array(6 * 1024 * 1024); // 6MB > 5MB limit
    mockFetchOnce(new Response(huge, { status: 200, headers: { "Content-Type": "image/png" } }));

    await expect(resolver.resolveAsset(VALID_URL))
      .rejects.toThrow(/ceiling/i);
  });

  it("rejects content whose magic bytes do not match the Content-Type", async () => {
    const badBytes = new Uint8Array([0x00, 0x00, 0x00, 0x00]);
    mockFetchOnce(new Response(badBytes, { status: 200, headers: { "Content-Type": "image/png" } }));

    await expect(resolver.resolveAsset(VALID_URL))
      .rejects.toThrow(/magic/i);
  });

  it("returns resolved bytes and metadata for a valid PNG", async () => {
    mockFetchOnce(new Response(TINY_PNG, { status: 200, headers: { "Content-Type": "image/png" } }));

    const asset = await resolver.resolveAsset(VALID_URL);
    expect(asset.mimeType).toBe("image/png");
    expect(asset.bytes.byteLength).toBe(TINY_PNG.byteLength);
    expect(asset.sizeBytes).toBe(TINY_PNG.byteLength);
  });

  it("rejects disallowed MIME types like image/svg+xml", async () => {
    mockFetchOnce(new Response(new Uint8Array([0x3c, 0x73, 0x76, 0x67]), {
      status: 200,
      headers: { "Content-Type": "image/svg+xml" },
    }));

    await expect(resolver.resolveAsset(VALID_URL))
      .rejects.toThrow(/mime/i);
  });
});
