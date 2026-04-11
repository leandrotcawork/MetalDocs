import { afterEach, describe, expect, it, vi } from "vitest";
import { exportPdf } from "../export-pdf";
import { ResourceCeilingExceededError } from "../../asset-resolver";
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

    const blob = await exportPdf("doc-1", "<p>Hi</p>");

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

    await exportPdf("doc-1", "<p>Hi</p>");

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

    await expect(exportPdf("doc-1", huge))
      .rejects.toBeInstanceOf(ResourceCeilingExceededError);
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("throws when backend returns non-2xx", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(
      new Response("forbidden", { status: 403 }),
    ));

    await expect(exportPdf("doc-1", "<p/>"))
      .rejects.toThrow(/PDF render failed/);
  });

  it("throws when Content-Type is not application/pdf", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(
      new Response("not a pdf", { status: 200, headers: { "Content-Type": "text/html" } }),
    ));

    await expect(exportPdf("doc-1", "<p/>"))
      .rejects.toThrow(/Content-Type/);
  });
});
