// Central registry of block types the MDDM engine renders.
// After Plan 2, every entry has all three renderers (React, toExternalHTML, DOCX).

export type BlockSupport = Readonly<{
  type: string;
  hasReactRender: boolean;
  hasExternalHtml: boolean;
  hasDocxEmitter: boolean;
}>;

export const BLOCK_REGISTRY: readonly BlockSupport[] = [
  // Standard BlockNote blocks
  { type: "paragraph",        hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "heading",          hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "bulletListItem",   hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "numberedListItem", hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "image",            hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "quote",            hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "divider",          hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },

  // MDDM custom blocks
  { type: "section",          hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "field",            hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "fieldGroup",       hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "repeatable",       hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "repeatableItem",   hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "richBlock",        hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "dataTable",        hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "dataTableRow",     hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "dataTableCell",    hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
];

export function getFullySupportedBlockTypes(): readonly string[] {
  return BLOCK_REGISTRY
    .filter((b) => b.hasReactRender && b.hasExternalHtml && b.hasDocxEmitter)
    .map((b) => b.type);
}
