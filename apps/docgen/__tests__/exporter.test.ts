import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

import { exportMDDMToDocx } from "../src/mddm/exporter.js";

describe("MDDM exporter", () => {
  it("renders a simple Section + Paragraph to DOCX bytes", async () => {
    const bytes = await exportMDDMToDocx({
      metadata: {
        document_code: "PO-118",
        title: "Test",
        revision_label: "REV01",
        mode: "production",
      },
      envelope: {
        mddm_version: 1,
        template_ref: null,
        blocks: [
          {
            id: "111",
            type: "section",
            props: {
              title: "Test",
            },
            children: [
              {
                id: "222",
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
    expect(bytes.length).toBeGreaterThan(1000);
  });

  it("renders all in-scope block types without throwing", async () => {
    const fixture = JSON.parse(
      readFileSync(
        join(__dirname, "../../../shared/schemas/test-fixtures/valid/full-block-types.json"),
        "utf8"
      )
    );
    const result = await exportMDDMToDocx({
      envelope: fixture,
      metadata: {
        document_code: "PO-TEST",
        title: "Test Document",
        revision_label: "REV01",
        mode: "debug",
      },
    });
    expect(result).toBeInstanceOf(Uint8Array);
    expect(result.length).toBeGreaterThan(0);
  });
});
