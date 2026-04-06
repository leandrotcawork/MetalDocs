import HTMLtoDOCX from "@turbodocx/html-to-docx";

type BrowserDocumentPayload = {
  documentCode?: string;
  title?: string;
  version?: string;
  html: string;
};

function invalid(code: string): never {
  throw new Error(code);
}

function normalizeBrowserPayload(input: unknown): BrowserDocumentPayload {
  if (typeof input !== "object" || input === null || Array.isArray(input)) {
    invalid("DOCGEN_INVALID_PAYLOAD");
  }

  const payload = input as Partial<BrowserDocumentPayload>;
  if (typeof payload.html !== "string" || !payload.html.trim()) {
    invalid("DOCGEN_INVALID_PAYLOAD");
  }

  return {
    documentCode: typeof payload.documentCode === "string" ? payload.documentCode.trim() : undefined,
    title: typeof payload.title === "string" ? payload.title.trim() : undefined,
    version: typeof payload.version === "string" ? payload.version.trim() : undefined,
    html: payload.html,
  };
}

export async function generateBrowserDocx(payload: unknown): Promise<Uint8Array> {
  const browserPayload = normalizeBrowserPayload(payload);
  const document = await HTMLtoDOCX(browserPayload.html);
  if (document instanceof Uint8Array) {
    return document;
  }
  if (document instanceof ArrayBuffer) {
    return new Uint8Array(document);
  }
  if (document instanceof Blob) {
    return new Uint8Array(await document.arrayBuffer());
  }
  return new Uint8Array(document);
}
