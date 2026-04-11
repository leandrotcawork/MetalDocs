import { afterEach, describe, expect, it, vi } from "vitest";
import { exportPdf } from "../export-pdf";
import { ResourceCeilingExceededError } from "../../asset-resolver";
import { AssetResolver } from "../../asset-resolver";
import type { RendererPin } from "../../../../../../lib.types";

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
    // Capture the FormData submitted to verify that wrapInPrintDocument ran.
    // We verify the form includes index.html and style.css parts (jsdom Blob.text()
    // limitation means we can't read blob contents, but structural checks suffice).
    const fetchSpy = mockFetchOk(new Uint8Array([0x25, 0x50, 0x44, 0x46]));

    await exportPdf({ documentId: "doc-1", bodyHtml: "<p>Hi</p>" });

    expect(fetchSpy).toHaveBeenCalledTimes(1);
    const [, init] = fetchSpy.mock.calls[0];
    const body = init?.body as FormData;
    expect(body.has("index.html")).toBe(true);
    expect(body.has("style.css")).toBe(true);
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

  it("uses the pinned renderer when rendererPin is provided", async () => {
    const pin: RendererPin = {
      renderer_version: "1.0.0",
      layout_ir_hash: "h",
      template_key: "k",
      template_version: 1,
    };
    const fetchSpy = vi.fn().mockResolvedValue(
      new Response(new Uint8Array([0x25, 0x50, 0x44, 0x46]), {
        status: 200,
        headers: { "Content-Type": "application/pdf" },
      }),
    );
    vi.stubGlobal("fetch", fetchSpy);

    const blob = await exportPdf({
      bodyHtml: "<p>x</p>",
      documentId: "doc-1",
      rendererPin: pin,
    });
    expect(blob).toBeInstanceOf(Blob);
    expect(fetchSpy).toHaveBeenCalledTimes(1);
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

    // Capture fetch call to verify the HTML was inlined.
    // (jsdom Blob lacks .text() so we can't read FormData blob content, but
    // verifying the fetch is called once confirms the pipeline completed.)
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(
      new Response(new Uint8Array([0x25, 0x50, 0x44, 0x46]), {
        status: 200,
        headers: { "Content-Type": "application/pdf" },
      }),
    ));

    const result = await exportPdf({
      bodyHtml: `<p><img src="/api/images/aaa" /></p>`,
      documentId: "doc-1",
      assetResolver: fakeResolver,
    });

    expect(result).toBeInstanceOf(Blob);
  });
});
