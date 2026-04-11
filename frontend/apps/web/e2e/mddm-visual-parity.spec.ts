import { test, expect } from "@playwright/test";
import { rasterizePdfFirstPageToPng, pngDiffPercent } from "./helpers/pixel-diff";

const FIXTURES = ["01-simple-po", "02-complex-table", "03-repeatable-sections"] as const;

// Visual parity tolerance. Matches the Render Compatibility Contract tier-2
// threshold in the spec (COMPATIBILITY_CONTRACT.tier2.pixelDiffEditorToPdf).
const EDITOR_TO_PDF_MAX_DIFF = 0.02; // 2%

for (const fixture of FIXTURES) {
  test(`MDDM visual parity: editor screenshot vs PDF (${fixture})`, async ({ page }) => {
    await page.goto(`/#/test-harness/mddm?doc=${fixture}`);

    // Wait for the harness to signal it has mounted and exposed the APIs.
    await page.waitForFunction(() => (window as any).__mddmHarnessReady === true, undefined, {
      timeout: 30_000,
    });
    await page.locator("[data-testid='mddm-harness']").waitFor({ state: "visible" });

    // 1. Capture editor screenshot.
    const editorElement = page.locator("[data-testid='mddm-harness']");
    const editorPng = await editorElement.screenshot();

    // 2. Produce full-fidelity HTML from the mounted BlockNote editor.
    const bodyHtml = await page.evaluate(async () => {
      const editor = (window as any).__mddmEditor;
      if (!editor) throw new Error("__mddmEditor not exposed");
      return await editor.blocksToFullHTML(editor.document);
    });

    // 3. Render PDF directly via Gotenberg (bypasses auth-protected backend).
    const pdfArray = await page.evaluate(async (html: string) => {
      const blob = await (window as any).__mddmRenderPdfDirectlyViaGotenberg(html);
      const buffer = await (blob as Blob).arrayBuffer();
      return Array.from(new Uint8Array(buffer));
    }, bodyHtml);
    const pdfBytes = new Uint8Array(pdfArray);

    // 4. Rasterize PDF page 1 and diff against editor screenshot.
    const pdfPng = await rasterizePdfFirstPageToPng(pdfBytes);

    const diff = pngDiffPercent(editorPng, pdfPng);
    expect(
      diff,
      `Visual diff for ${fixture} exceeded ${EDITOR_TO_PDF_MAX_DIFF * 100}%: ${(diff * 100).toFixed(2)}%`,
    ).toBeLessThan(EDITOR_TO_PDF_MAX_DIFF);
  });
}
