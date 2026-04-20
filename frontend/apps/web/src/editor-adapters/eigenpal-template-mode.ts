import type { BlockContent, Paragraph } from "@eigenpal/docx-js-editor/core";

export type PlaceholderRun = { type: "placeholder"; id: string; label: string };
export type EditableZone = { id: string; label: string };
export type BlockNode = BlockContent;

type EigenpalInlineRunNode = Paragraph["content"][number];
type EigenpalInlineSdtNode = Extract<EigenpalInlineRunNode, { type: "inlineSdt" }>;

const PLACEHOLDER_TAG_PREFIX = "placeholder:";
const ZONE_START_BOOKMARK_PREFIX = "zone-start:";
const MAX_BOOKMARK_ID = 2147483646;

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

function createZoneBookmarkId(zoneId: string): number {
  let hash = 0;
  for (const char of zoneId) {
    hash = ((hash * 31) + char.charCodeAt(0)) >>> 0;
  }
  return (hash % MAX_BOOKMARK_ID) + 1;
}

function buildZoneStartMarkerParagraph(zoneId: string, bookmarkId: number): Paragraph {
  return {
    type: "paragraph",
    content: [
      {
        type: "bookmarkStart",
        id: bookmarkId,
        name: `${ZONE_START_BOOKMARK_PREFIX}${zoneId}`,
      },
    ],
  };
}

function buildZoneEndMarkerParagraph(bookmarkId: number): Paragraph {
  return {
    type: "paragraph",
    content: [
      {
        type: "bookmarkEnd",
        id: bookmarkId,
      },
    ],
  };
}

function readZoneStartMarker(paragraph: Paragraph): { zoneId: string; bookmarkId: number } | null {
  const marker = paragraph.content.find((node) => node.type === "bookmarkStart");
  if (!marker || !marker.name.startsWith(ZONE_START_BOOKMARK_PREFIX)) {
    return null;
  }

  const zoneId = marker.name.slice(ZONE_START_BOOKMARK_PREFIX.length).trim();
  if (!zoneId) {
    return null;
  }

  return { zoneId, bookmarkId: marker.id };
}

function hasMatchingZoneEndMarker(paragraph: Paragraph, bookmarkId: number): boolean {
  return paragraph.content.some((node) => node.type === "bookmarkEnd" && node.id === bookmarkId);
}

export function wrapZone(zone: EditableZone, blocks: BlockNode[]): BlockNode[] {
  const id = zone.id.trim();
  if (!id) {
    throw new Error("Zone id is required.");
  }

  const bookmarkId = createZoneBookmarkId(id);
  return [buildZoneStartMarkerParagraph(id, bookmarkId), ...blocks, buildZoneEndMarkerParagraph(bookmarkId)];
}

export function extractZones(content: BlockNode[]): Array<{
  zone: EditableZone;
  startIndex: number;
  endIndex: number;
  blocks: BlockNode[];
}> {
  const zones: Array<{
    zone: EditableZone;
    startIndex: number;
    endIndex: number;
    blocks: BlockNode[];
  }> = [];

  for (let index = 0; index < content.length; index += 1) {
    const candidate = content[index];
    if (!candidate || candidate.type !== "paragraph") {
      continue;
    }

    const startMarker = readZoneStartMarker(candidate);
    if (!startMarker) {
      continue;
    }

    const endIndex = content.findIndex(
      (node, nextIndex) =>
        nextIndex > index &&
        node.type === "paragraph" &&
        hasMatchingZoneEndMarker(node, startMarker.bookmarkId),
    );

    if (endIndex === -1) {
      continue;
    }

    zones.push({
      zone: { id: startMarker.zoneId, label: startMarker.zoneId },
      startIndex: index,
      endIndex,
      blocks: content.slice(index + 1, endIndex),
    });

    index = endIndex;
  }

  return zones;
}
