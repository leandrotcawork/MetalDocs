import type { Paragraph } from "@eigenpal/docx-js-editor";

export type PlaceholderRun = { type: "placeholder"; id: string; label: string };

type EigenpalInlineRunNode = Paragraph["content"][number];
type EigenpalInlineSdtNode = Extract<EigenpalInlineRunNode, { type: "inlineSdt" }>;

const PLACEHOLDER_TAG_PREFIX = "placeholder:";

export function placeholderToRun(p: PlaceholderRun): EigenpalInlineSdtNode {
  return {
    type: "inlineSdt",
    properties: {
      sdtType: "richText",
      tag: `${PLACEHOLDER_TAG_PREFIX}${p.id}`,
      alias: p.label,
      placeholder: p.label,
    },
    content: [
      {
        type: "run",
        content: [{ type: "text", text: p.label }],
      },
    ],
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
