// SPIKE RED: @eigenpal/docx-js-editor serializes `blockSdt` to `<w:sdt>` but
// parseDocx does not re-emit `blockSdt` nodes — block SDT markers are lost on
// DOCX round-trip. Confirmed by grep: `blockSdt` appears once (serializer
// chunk-37SLIJPH.mjs) and zero times in parser chunks. Missing API: block-SDT
// round-trip (parser-side block-level SDT node emission).
import { describe, expect, it } from "vitest";
import {
  createEmptyDocument,
  DocumentAgent,
  parseDocx,
  type BlockContent,
  type Paragraph,
  type Run,
} from "@eigenpal/docx-js-editor/core";

import { extractZone, wrapZone } from "../eigenpal-template-mode";

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

describe("eigenpal zone spike", () => {
  it("round-trips block zone marker and keeps wrapped paragraph inside", async () => {
    const documentModel = createEmptyDocument();
    const blocks: BlockContent[] = [
      paragraphWithText("Paragraph 1"),
      paragraphWithText("Paragraph 2"),
      paragraphWithText("Paragraph 3"),
    ];

    const [paragraph1, paragraph2, paragraph3] = blocks;
    if (!paragraph1 || !paragraph2 || !paragraph3) {
      throw new Error("Expected three paragraph blocks.");
    }

    documentModel.package.document.content = [
      paragraph1,
      wrapZone("observations", [paragraph2]),
      paragraph3,
    ];

    const docxBytes = await DocumentAgent.fromDocument(documentModel).toBuffer();
    const reparsed = await parseDocx(docxBytes);

    const [reparsedFirst, reparsedZone, reparsedThird] = reparsed.package.document.content;
    if (!reparsedFirst || reparsedFirst.type !== "paragraph") {
      throw new Error("Expected first block to be a paragraph.");
    }
    if (!reparsedZone || reparsedZone.type !== "blockSdt") {
      throw new Error("Expected second block to be a blockSdt zone marker.");
    }
    if (!reparsedThird || reparsedThird.type !== "paragraph") {
      throw new Error("Expected third block to be a paragraph.");
    }

    expect(paragraphText(reparsedFirst)).toBe("Paragraph 1");
    expect(paragraphText(reparsedThird)).toBe("Paragraph 3");

    const extracted = extractZone(reparsedZone);
    expect(extracted).not.toBeNull();
    expect(extracted?.zone.id).toBe("observations");
    expect(extracted?.blocks).toHaveLength(1);

    const wrappedParagraph = extracted?.blocks[0];
    if (!wrappedParagraph || wrappedParagraph.type !== "paragraph") {
      throw new Error("Expected wrapped block to be a paragraph.");
    }
    expect(paragraphText(wrappedParagraph)).toBe("Paragraph 2");
  });
});
