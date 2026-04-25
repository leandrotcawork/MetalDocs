import type { BlockContent, Paragraph } from "@eigenpal/docx-js-editor/core";

export type PlaceholderRun = {
  type: "placeholder";
  id: string;
  label: string;
  placeholderType?: "text" | "date" | "number" | "select" | "user" | "picture" | "computed";
  options?: string[];
};
export type BlockNode = BlockContent;

type EigenpalInlineRunNode = Paragraph["content"][number];
type EigenpalInlineSdtNode = Extract<EigenpalInlineRunNode, { type: "inlineSdt" }>;
type EigenpalBlockSdtNode = Extract<BlockNode, { type: "blockSdt" }>;

const PLACEHOLDER_TAG_PREFIX = "placeholder:";

export function placeholderToRun(p: PlaceholderRun): EigenpalInlineSdtNode {
  const sdtType =
    p.placeholderType === "date"
      ? "date"
      : p.placeholderType === "select"
        ? "dropdown"
        : p.placeholderType === "picture"
          ? "picture"
          : p.placeholderType === "computed"
            ? "plainText"
            : "richText";
  const props: any = {
    sdtType,
    tag: `${PLACEHOLDER_TAG_PREFIX}${p.id}`,
    alias: p.label,
    placeholder: p.label,
  };
  if (p.placeholderType === "select" && p.options) {
    props.listItems = p.options.map((o) => ({ displayText: o, value: o }));
  }
  if (p.placeholderType === "computed") {
    props.lock = "sdtContentLocked";
  }

  return {
    type: "inlineSdt",
    properties: props,
    content: [{ type: "run", content: [{ type: "text", text: p.label }] }],
  };
}

export function wrapFrozenContent(blocks: BlockNode[]): EigenpalBlockSdtNode {
  return {
    type: "blockSdt",
    properties: {
      sdtType: "richText",
      lock: "sdtContentLocked",
    },
    content: blocks,
  };
}

export function runToPlaceholder(node: EigenpalInlineRunNode): PlaceholderRun | null {
  if (node.type !== "inlineSdt") {
    return null;
  }

  const tag = node.properties.tag;
  if (!tag?.startsWith(PLACEHOLDER_TAG_PREFIX)) {
    return null;
  }

  const id = tag.slice(PLACEHOLDER_TAG_PREFIX.length).trim();
  if (!id) {
    return null;
  }

  const label =
    node.properties.alias?.trim() ||
    node.content
      .flatMap((child) =>
        child.type === "run" ? child.content : child.children.flatMap((nested) => (nested.type === "run" ? nested.content : [])),
      )
      .filter((content) => content.type === "text")
      .map((content) => content.text)
      .join("")
      .trim() ||
    id;

  return {
    type: "placeholder",
    id,
    label,
  };
}
