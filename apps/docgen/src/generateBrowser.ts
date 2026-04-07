import HTMLtoDOCX from "@turbodocx/html-to-docx";

type BrowserDocumentMargins = {
  top: number;
  right: number;
  bottom: number;
  left: number;
};

type BrowserDocumentPayload = {
  documentCode?: string;
  title?: string;
  version?: string;
  html: string;
  margins?: BrowserDocumentMargins;
};

function invalid(code: string): never {
  throw new Error(code);
}

function inchesToTwips(inches: number): number {
  return Math.round(inches * 1440);
}

function parseBrowserMargins(raw: unknown): BrowserDocumentMargins | undefined {
  if (typeof raw !== "object" || raw === null || Array.isArray(raw)) {
    return undefined;
  }
  const m = raw as Record<string, unknown>;
  if (
    typeof m.top !== "number" ||
    typeof m.right !== "number" ||
    typeof m.bottom !== "number" ||
    typeof m.left !== "number"
  ) {
    return undefined;
  }
  return { top: m.top, right: m.right, bottom: m.bottom, left: m.left };
}

function normalizeBrowserPayload(input: unknown): BrowserDocumentPayload {
  if (typeof input !== "object" || input === null || Array.isArray(input)) {
    invalid("DOCGEN_INVALID_PAYLOAD");
  }

  const payload = input as Partial<BrowserDocumentPayload> & Record<string, unknown>;
  if (typeof payload.html !== "string" || !payload.html.trim()) {
    invalid("DOCGEN_INVALID_PAYLOAD");
  }

  return {
    documentCode: typeof payload.documentCode === "string" ? payload.documentCode.trim() : undefined,
    title: typeof payload.title === "string" ? payload.title.trim() : undefined,
    version: typeof payload.version === "string" ? payload.version.trim() : undefined,
    html: payload.html,
    margins: parseBrowserMargins(payload.margins),
  };
}

export async function generateBrowserDocx(payload: unknown): Promise<Uint8Array> {
  const browserPayload = normalizeBrowserPayload(payload);

  let document: Uint8Array | ArrayBuffer | Blob | Buffer;
  if (browserPayload.margins) {
    const { top, right, bottom, left } = browserPayload.margins;
    document = await HTMLtoDOCX(browserPayload.html, null, {
      margins: {
        top: inchesToTwips(top),
        right: inchesToTwips(right),
        bottom: inchesToTwips(bottom),
        left: inchesToTwips(left),
      },
    });
  } else {
    document = await HTMLtoDOCX(browserPayload.html);
  }

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
