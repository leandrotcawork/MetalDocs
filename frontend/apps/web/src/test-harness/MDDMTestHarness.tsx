import { useEffect, useState } from "react";
import { MDDMEditor } from "../features/documents/mddm-editor/MDDMEditor";
import { mddmToBlockNote, type MDDMEnvelope } from "../features/documents/mddm-editor/adapter";
import { exportDocx } from "../features/documents/mddm-editor/engine/export";
import { PRINT_STYLESHEET, wrapInPrintDocument } from "../features/documents/mddm-editor/engine/print-stylesheet";

// Dev-only: loads a golden fixture by name and exposes export APIs to Playwright.
// This component is only reachable via App.tsx when import.meta.env.DEV is true.

const FIXTURES: Record<string, () => Promise<MDDMEnvelope>> = {
  "01-simple-po": () =>
    import("../features/documents/mddm-editor/engine/golden/fixtures/01-simple-po/input.mddm.json").then(
      (m) => m.default as unknown as MDDMEnvelope,
    ),
  "02-complex-table": () =>
    import("../features/documents/mddm-editor/engine/golden/fixtures/02-complex-table/input.mddm.json").then(
      (m) => m.default as unknown as MDDMEnvelope,
    ),
  "03-repeatable-sections": () =>
    import("../features/documents/mddm-editor/engine/golden/fixtures/03-repeatable-sections/input.mddm.json").then(
      (m) => m.default as unknown as MDDMEnvelope,
    ),
};

async function renderPdfDirectlyViaGotenberg(bodyHtml: string): Promise<Blob> {
  const fullHtml = wrapInPrintDocument(bodyHtml);
  const form = new FormData();
  form.append("files", new Blob([fullHtml], { type: "text/html" }), "index.html");
  form.append("files", new Blob([PRINT_STYLESHEET], { type: "text/css" }), "style.css");

  const response = await fetch("/__gotenberg/forms/chromium/convert/html", {
    method: "POST",
    body: form,
  });
  if (!response.ok) {
    throw new Error(`Gotenberg render failed: ${response.status}`);
  }
  const arrayBuffer = await response.arrayBuffer();
  return new Blob([arrayBuffer], { type: "application/pdf" });
}

export function MDDMTestHarness() {
  const [envelope, setEnvelope] = useState<MDDMEnvelope | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!import.meta.env.DEV) {
      setError("Test harness is disabled in production builds.");
      return;
    }
    const params = new URLSearchParams(window.location.hash.split("?")[1] ?? "");
    const docName = params.get("doc");
    if (!docName || !FIXTURES[docName]) {
      setError(`Unknown fixture: ${docName ?? "(none)"}`);
      return;
    }
    FIXTURES[docName]!().then(setEnvelope).catch((err) => setError(String(err)));
  }, []);

  useEffect(() => {
    if (!envelope) return;
    (window as any).__mddmExportDocx = () => exportDocx(envelope, { rendererPin: null });
    (window as any).__mddmRenderPdfDirectlyViaGotenberg = renderPdfDirectlyViaGotenberg;
    (window as any).__mddmHarnessReady = true;
  }, [envelope]);

  if (error) return <div data-testid="harness-error">{error}</div>;
  if (!envelope) return <div data-testid="harness-loading">Loading&#8230;</div>;

  const blocks = mddmToBlockNote(envelope);

  return (
    <div data-testid="mddm-harness">
      <MDDMEditor
        initialContent={blocks as any}
        readOnly={true}
      />
    </div>
  );
}
