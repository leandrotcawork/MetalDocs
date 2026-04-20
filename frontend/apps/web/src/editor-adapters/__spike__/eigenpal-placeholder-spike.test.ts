import { describe, expect, it } from "vitest";
import {
  createEmptyDocument,
  DocumentAgent,
  parseDocx,
  serializeDocx,
} from "@eigenpal/docx-js-editor/core";

import {
  placeholderToRun,
  runToPlaceholder,
  type PlaceholderRun,
} from "../eigenpal-template-mode";

describe("eigenpal placeholder spike", () => {
  it("round-trips placeholder chip inline node id through DOCX bytes", async () => {
    const placeholder: PlaceholderRun = {
      type: "placeholder",
      id: "customer_name",
      label: "Customer Name",
    };

    const documentModel = createEmptyDocument();
    const firstBlock = documentModel.package.document.content[0];
    if (!firstBlock || firstBlock.type !== "paragraph") {
      throw new Error("Expected createEmptyDocument() to include at least one paragraph.");
    }

    firstBlock.content = [placeholderToRun(placeholder)];

    const documentXml = serializeDocx(documentModel);
    expect(documentXml).toContain("placeholder:customer_name");

    const docxBytes = await DocumentAgent.fromDocument(documentModel).toBuffer();
    const reparsed = await parseDocx(docxBytes);

    const reparsedFirstBlock = reparsed.package.document.content[0];
    if (!reparsedFirstBlock || reparsedFirstBlock.type !== "paragraph") {
      throw new Error("Reparsed DOCX did not contain a paragraph.");
    }

    const reparsedPlaceholder = reparsedFirstBlock.content
      .filter((node): node is Parameters<typeof runToPlaceholder>[0] =>
        node.type === "inlineSdt",
      )
      .map((node) => runToPlaceholder(node))
      .find((value): value is PlaceholderRun => value !== null);

    expect(reparsedPlaceholder).toEqual({
      type: "placeholder",
      id: "customer_name",
      label: "Customer Name",
    });
  });
});
