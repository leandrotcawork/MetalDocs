import JSZip from "jszip";
import { describe, expect, it } from "vitest";

import { exportMDDMToDocx } from "../src/mddm/exporter.js";

describe("exportMDDMToDocx", () => {
  it("exports a section with a paragraph into DOCX bytes", async () => {
    const bytes = await exportMDDMToDocx({
      metadata: {
        document_code: "DOC-001",
        title: "Plan snippet",
        revision_label: "R1",
        mode: "debug",
      },
      envelope: {
        mddm_version: 1,
        template_ref: null,
        blocks: [
          {
            id: "section-1",
            type: "section",
            props: {
              title: "Plan snippet",
            },
            children: [
              {
                id: "paragraph-1",
                type: "paragraph",
                props: {},
                children: [
                  {
                    text: "Hello from the fixture.",
                  },
                ],
              },
            ],
          },
        ],
      },
    });

    expect(bytes[0]).toBe(0x50);
    expect(bytes[1]).toBe(0x4b);

    const zip = await JSZip.loadAsync(bytes);
    const documentXml = await zip.file("word/document.xml")?.async("string");

    expect(documentXml).toBeDefined();
    expect(documentXml).toContain("1. Plan snippet");
    expect(documentXml).toContain("Hello from the fixture.");
  });
});
