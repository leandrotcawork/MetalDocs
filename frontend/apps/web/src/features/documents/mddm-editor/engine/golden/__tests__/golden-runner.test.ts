import { describe, expect, it } from "vitest";
import { readFileSync, existsSync } from "node:fs";
import { resolve } from "node:path";
import { mddmToDocx } from "../../docx-emitter";
import { defaultLayoutTokens } from "../../layout-ir";
import { normalizeDocxXml, unzipDocxDocumentXml } from "../golden-helpers";
import type { MDDMEnvelope } from "../../../adapter";

const FIXTURE_DIR = resolve(__dirname, "../fixtures/01-simple-po");
const INPUT_PATH = resolve(FIXTURE_DIR, "input.mddm.json");
const EXPECTED_DOCX_XML = resolve(FIXTURE_DIR, "expected.document.xml");

describe("Golden fixture: 01-simple-po", () => {
  it("emits DOCX matching the approved document.xml", async () => {
    const envelope = JSON.parse(readFileSync(INPUT_PATH, "utf8")) as MDDMEnvelope;
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    const xml = await unzipDocxDocumentXml(blob);
    const actual = normalizeDocxXml(xml);

    if (!existsSync(EXPECTED_DOCX_XML)) {
      throw new Error(
        `Golden file missing: ${EXPECTED_DOCX_XML}\n\nGenerate it once with:\n  MDDM_GOLDEN_UPDATE=1 npx vitest run <generate-golden.test.ts>\nThen commit the file after manual review.`,
      );
    }

    const expected = normalizeDocxXml(readFileSync(EXPECTED_DOCX_XML, "utf8"));
    expect(actual).toBe(expected);
  });
});
