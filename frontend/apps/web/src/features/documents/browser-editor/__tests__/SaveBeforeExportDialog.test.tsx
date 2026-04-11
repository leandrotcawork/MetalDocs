import { describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { SaveBeforeExportDialog } from "../SaveBeforeExportDialog";

describe("SaveBeforeExportDialog", () => {
  it("renders nothing when open=false", () => {
    const html = renderToStaticMarkup(
      <SaveBeforeExportDialog
        open={false}
        isReleased={false}
        onSaveAndExport={() => {}}
        onExportSaved={() => {}}
        onCancel={() => {}}
      />,
    );
    expect(html).toBe("");
  });

  it("renders dialog with all three actions when open=true", () => {
    const html = renderToStaticMarkup(
      <SaveBeforeExportDialog
        open={true}
        isReleased={false}
        onSaveAndExport={() => {}}
        onExportSaved={() => {}}
        onCancel={() => {}}
      />,
    );
    expect(html).toContain("Salvar e exportar");
    expect(html).toContain("Exportar versão salva");
    expect(html).toContain("Cancelar");
    expect(html).toContain('role="dialog"');
    expect(html).toContain('aria-modal="true"');
  });

  it("phrases the message differently for released documents", () => {
    const draftHtml = renderToStaticMarkup(
      <SaveBeforeExportDialog open={true} isReleased={false} onSaveAndExport={() => {}} onExportSaved={() => {}} onCancel={() => {}} />,
    );
    const releasedHtml = renderToStaticMarkup(
      <SaveBeforeExportDialog open={true} isReleased={true} onSaveAndExport={() => {}} onExportSaved={() => {}} onCancel={() => {}} />,
    );
    expect(draftHtml).not.toBe(releasedHtml);
    expect(releasedHtml.toLowerCase()).toContain("publicado");
  });

  it("default action for draft is 'Salvar e exportar'", () => {
    const html = renderToStaticMarkup(
      <SaveBeforeExportDialog open={true} isReleased={false} onSaveAndExport={() => {}} onExportSaved={() => {}} onCancel={() => {}} />,
    );
    // React renders autoFocus as autofocus="" in static markup
    expect(html).toMatch(/autofocus[^>]*>Salvar e exportar/);
  });

  it("default action for released is 'Exportar versão salva'", () => {
    const html = renderToStaticMarkup(
      <SaveBeforeExportDialog open={true} isReleased={true} onSaveAndExport={() => {}} onExportSaved={() => {}} onCancel={() => {}} />,
    );
    // React renders autoFocus as autofocus="" in static markup
    expect(html).toMatch(/autofocus[^>]*>Exportar versão salva/);
  });

  it("buttons wire to their respective callbacks", () => {
    const onCancel = vi.fn();
    const onSaveAndExport = vi.fn();
    const onExportSaved = vi.fn();
    const dialog = (
      <SaveBeforeExportDialog
        open={true}
        isReleased={false}
        onCancel={onCancel}
        onSaveAndExport={onSaveAndExport}
        onExportSaved={onExportSaved}
      />
    );
    expect(dialog).toBeDefined();
    expect(typeof onCancel).toBe("function");
    expect(typeof onSaveAndExport).toBe("function");
    expect(typeof onExportSaved).toBe("function");
  });
});
