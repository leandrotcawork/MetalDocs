// Plan 1 exports (retained)
export { SectionExternalHTML, type SectionExternalHTMLProps } from "./section-html";
export { FieldExternalHTML, type FieldExternalHTMLProps } from "./field-html";
export { FieldGroupExternalHTML, type FieldGroupExternalHTMLProps } from "./field-group-html";

// Plan 2 additions: only inline-content blocks get custom toExternalHTML.
// DataTableCell is the only content:"inline" block added in Plan 2.
// Repeatable, RepeatableItem, RichBlock, DataTable, DataTableRow are
// content:"none" and rely on BlockNote's render() fallback.
export { DataTableCellExternalHTML, type DataTableCellExternalHTMLProps } from "./data-table-cell-html";
