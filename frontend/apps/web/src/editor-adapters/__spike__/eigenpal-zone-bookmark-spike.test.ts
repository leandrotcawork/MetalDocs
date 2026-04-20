import { describe, expect, it } from "vitest";
import {
  createEmptyDocument,
  DocumentAgent,
  parseDocx,
  type BlockContent,
  type BookmarkEnd,
  type BookmarkStart,
  type Paragraph,
  type Run,
} from "@eigenpal/docx-js-editor/core";

import { extractZones, wrapZone, type EditableZone } from "../eigenpal-template-mode";

function paragraphWithText(text: string): Paragraph {
  return {
    type: "paragraph",
    content: [
      {
        type: "run",
        content: [{ type: "text", text }],
      } satisfies Run,
    ],
  };
}

function paragraphText(paragraph: Paragraph): string {
  return paragraph.content
    .filter((node): node is Run => node.type === "run")
    .flatMap((node) => node.content)
    .filter((node) => node.type === "text")
    .map((node) => node.text)
    .join("");
}

function getBookmarkStart(paragraph: Paragraph): BookmarkStart | null {
  const starts = paragraph.content.filter(
    (node): node is BookmarkStart => node.type === "bookmarkStart",
  );
  return starts[0] ?? null;
}

function getBookmarkEnd(paragraph: Paragraph): BookmarkEnd | null {
  const ends = paragraph.content.filter(
    (node): node is BookmarkEnd => node.type === "bookmarkEnd",
  );
  return ends[0] ?? null;
}

describe("eigenpal zone bookmark spike", () => {
  it("round-trips bookmark marker paragraphs and reconstructs zone boundaries", async () => {
    const zone: EditableZone = { id: "observations", label: "Observations" };

    const paragraph1 = paragraphWithText("Paragraph 1");
    const paragraph2 = paragraphWithText("Paragraph 2");
    const paragraph3 = paragraphWithText("Paragraph 3");

    const documentModel = createEmptyDocument();
    const wrappedZoneBlocks = wrapZone(zone, [paragraph2]);

    documentModel.package.document.content = [
      paragraph1,
      ...wrappedZoneBlocks,
      paragraph3,
    ] satisfies BlockContent[];

    const docxBytes = await DocumentAgent.fromDocument(documentModel).toBuffer();
    const reparsed = await parseDocx(docxBytes);

    const reparsedBlocks = reparsed.package.document.content;
    const extractedZones = extractZones(reparsedBlocks);

    expect(extractedZones).toHaveLength(1);
    const extractedZone = extractedZones[0];
    if (!extractedZone) {
      throw new Error("Expected one extracted zone.");
    }

    expect(extractedZone.zone.id).toBe("observations");
    expect(extractedZone.blocks).toHaveLength(1);

    const wrappedBlock = extractedZone.blocks[0];
    if (!wrappedBlock || wrappedBlock.type !== "paragraph") {
      throw new Error("Expected extracted wrapped block to be a paragraph.");
    }
    expect(paragraphText(wrappedBlock)).toBe("Paragraph 2");

    const startMarkerBlock = reparsedBlocks[extractedZone.startIndex];
    if (!startMarkerBlock || startMarkerBlock.type !== "paragraph") {
      throw new Error("Expected start marker block to be a paragraph.");
    }

    const bookmarkStart = getBookmarkStart(startMarkerBlock);
    expect(bookmarkStart).not.toBeNull();
    expect(bookmarkStart?.name).toBe("zone-start:observations");

    const endMarkerBlock = reparsedBlocks[extractedZone.endIndex];
    if (!endMarkerBlock || endMarkerBlock.type !== "paragraph") {
      throw new Error("Expected end marker block to be a paragraph.");
    }

    const bookmarkEnd = getBookmarkEnd(endMarkerBlock);
    expect(bookmarkEnd).not.toBeNull();
    expect(bookmarkEnd?.id).toBe(bookmarkStart?.id);
  });
});
