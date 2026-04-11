export type BlockSupport = Readonly<{
  type: string;
  hasReactRender: boolean;
  hasExternalHtml: boolean;
  hasDocxEmitter: boolean;
}>;

export const BLOCK_REGISTRY: readonly BlockSupport[] = [
  { type: "paragraph",     hasReactRender: true, hasExternalHtml: true,  hasDocxEmitter: true },
  { type: "heading",       hasReactRender: true, hasExternalHtml: true,  hasDocxEmitter: true },
  { type: "section",       hasReactRender: true, hasExternalHtml: true,  hasDocxEmitter: true },
  { type: "field",         hasReactRender: true, hasExternalHtml: true,  hasDocxEmitter: true },
  { type: "fieldGroup",    hasReactRender: true, hasExternalHtml: true,  hasDocxEmitter: true },
  { type: "repeatable",     hasReactRender: true, hasExternalHtml: false, hasDocxEmitter: false },
  { type: "repeatableItem", hasReactRender: true, hasExternalHtml: false, hasDocxEmitter: false },
  { type: "richBlock",      hasReactRender: true, hasExternalHtml: false, hasDocxEmitter: false },
  { type: "dataTable",      hasReactRender: true, hasExternalHtml: false, hasDocxEmitter: false },
  { type: "dataTableRow",   hasReactRender: true, hasExternalHtml: false, hasDocxEmitter: false },
  { type: "dataTableCell",  hasReactRender: true, hasExternalHtml: false, hasDocxEmitter: false },
];

export function getFullySupportedBlockTypes(): readonly string[] {
  return BLOCK_REGISTRY
    .filter((b) => b.hasReactRender && b.hasExternalHtml && b.hasDocxEmitter)
    .map((b) => b.type);
}
