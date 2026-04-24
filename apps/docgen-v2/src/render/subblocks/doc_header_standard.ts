import { SubBlockRenderer, SubBlockContext } from "./registry";

function str(v: unknown): string {
  if (v === null || v === undefined) return "";
  return String(v);
}

function cell(text: string): string {
  return `<w:tc><w:p><w:r><w:t xml:space="preserve">${text}</w:t></w:r></w:p></w:tc>`;
}

function row(label: string, value: string): string {
  return `<w:tr>${cell(label)}${cell(value)}</w:tr>`;
}

export const DocHeaderStandard: SubBlockRenderer = {
  key: "doc_header_standard",
  async render(ctx: SubBlockContext): Promise<string> {
    const v = ctx.values;
    const title = str(v.title);
    const docCode = str(v.doc_code);
    const effectiveDate = str(v.effective_date);
    const revisionNumber = str(v.revision_number);

    return (
      `<w:p><w:r><w:t xml:space="preserve">${title}</w:t></w:r></w:p>` +
      `<w:tbl>` +
      row("Doc Code", docCode) +
      row("Effective Date", effectiveDate) +
      row("Revision", revisionNumber) +
      `</w:tbl>`
    );
  },
};
