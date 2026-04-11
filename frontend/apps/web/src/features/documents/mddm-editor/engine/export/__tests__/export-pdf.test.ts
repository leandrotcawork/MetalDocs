import { afterEach, describe, expect, it, vi } from "vitest";
import { exportPdf } from "../export-pdf";
import { ResourceCeilingExceededError } from "../../asset-resolver";
import { AssetResolver } from "../../asset-resolver";
import * as wrapModule from "../../print-stylesheet/wrap-print-document";

function mockFetchOk(pdfBytes: Uint8Array): ReturnType<typeof vi.fn> {
  const spy = vi.fn().mockResolvedValue(
    new Response(pdfBytes.buffer as ArrayBuffer, {
      status: 200,
      headers: { "Content-Type": "application/pdf" },
    }),
  );
  vi.stubGlobal("fetch", spy);
  return spy;
}

describe("exportPdf", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("POSTs multipart/form-data to /api/v1/documents/{id}/render/pdf", async () => {
    const fetchSpy = mockFetchOk(new Uint8Array([0x25, 0x50, 0x44, 0x46])); // "%PDF"

    const blob = await exportPdf({ documentId: "doc-1", bodyHtml: "<p>Hi</p>" });

    expect(blob.type).toBe("application/pdf");
    expect(fetchSpy).toHaveBeenCalledTimes(1);

    const [url, init] = fetchSpy.mock.calls[0];
    expect(url).toBe("/api/v1/documents/doc-1/render/pdf");
    expect(init?.method).toBe("POST");
    expect(init?.credentials).toBe("same-origin");

    const body = init?.body as FormData;
    expect(body).toBeInstanceOf(FormData);
    expect(body.has("index.html")).toBe(true);
    expect(body.has("style.css")).toBe(true);
  });

  it("wraps the body HTML in a full print document", async () => {
    // jsdom's FormData.get() returns a File without .text()/.arrayBuffer(),
    // so we spy on wrapInPrintDocument to verify the HTML content it receives.
    const wrapSpy = vi.spyOn(wrapModule, "wrapInPrintDocument");
    mockFetchOk(new Uint8Array([0x25, 0x50, 0x44, 0x46]));

    await exportPdf({ documentId: "doc-1", bodyHtml: "<p>Hi</p>" });

    expect(wrapSpy).toHaveBeenCalledWith("<p>Hi</p>");
    const htmlText = wrapSpy.mock.results[0].value as string;
    expect(htmlText).toContain("<!DOCTYPE html>");
    expect(htmlText).toContain("<p>Hi</p>");
    expect(htmlText).toContain("Carlito");

    wrapSpy.mockRestore();
  });

  it("throws ResourceCeilingExceededError when payload exceeds maxHtmlPayloadBytes", async () => {
    const huge = "x".repeat(11 * 1024 * 1024);
    const fetchSpy = mockFetchOk(new Uint8Array([0x25, 0x50, 0x44, 0x46]));

    await expect(exportPdf({ documentId: "doc-1", bodyHtml: huge }))
      .rejects.toBeInstanceOf(ResourceCeilingExceededError);
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("throws when backend returns non-2xx", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(
      new Response("forbidden", { status: 403 }),
    ));

    await expect(exportPdf({ documentId: "doc-1", bodyHtml: "<p/>" }))
      .rejects.toThrow(/PDF render failed/);
  });

  it("throws when Content-Type is not application/pdf", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(
      new Response("not a pdf", { status: 200, headers: { "Content-Type": "text/html" } }),
    ));

    await expect(exportPdf({ documentId: "doc-1", bodyHtml: "<p/>" }))
      .rejects.toThrow(/Content-Type/);
  });

  it("URL-encodes documentId with special characters", async () => {
    const fetchMock = mockFetchOk(new Uint8Array([0x25, 0x50, 0x44, 0x46]));
    await exportPdf({ documentId: "doc/evil?q=1", bodyHtml: "<p>test</p>" });
    const [url] = fetchMock.mock.calls[0];
    expect(url).toContain("doc%2Fevil%3Fq%3D1");
  });
});

describe("exportPdf asset inlining", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("inlines image URLs as data: URIs in the HTML sent to Gotenberg", async () => {
    const PNG = new Uint8Array([0x89, 0x50, 0x4e, 0x47]);
    const fakeResolver = {
      async resolveAsset(_url: string) {
        return { bytes: PNG, mimeType: "image/png" as const, sizeBytes: PNG.byteLength };
      },
    } as unknown as AssetResolver;

    // Spy on wrapInPrintDocument to capture the inlined HTML before it's put
    // into a Blob (jsdom Blob lacks .text(), so we can't read from FormData).
    const wrapSpy = vi.spyOn(wrapModule, "wrapInPrintDocument");

    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(
      new Response(new Uint8Array([0x25, 0x50, 0x44, 0x46]), {
        status: 200,
        headers: { "Content-Type": "application/pdf" },
      }),
    ));

    await exportPdf({
      bodyHtml: `<p><img src="/api/images/aaa" /></p>`,
      documentId: "doc-1",
      assetResolver: fakeResolver,
    });

    // wrapInPrintDocument receives the already-inlined body
    const inlinedBody = wrapSpy.mock.calls[0][0] as string;
    expect(inlinedBody).toContain("data:image/png;base64,");
    expect(inlinedBody).not.toContain("/api/images/aaa");

    wrapSpy.mockRestore();
  });
});
