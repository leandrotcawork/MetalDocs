import { describe, expect, it } from "vitest";
import { readFileSync, existsSync } from "node:fs";
import { resolve } from "node:path";
import { mddmToDocx } from "../../docx-emitter";
import { defaultLayoutTokens } from "../../layout-ir";
import { normalizeDocxXml, unzipDocxDocumentXml } from "../golden-helpers";
import type { MDDMEnvelope } from "../../../adapter";

const FIXTURE = resolve(__dirname, "../fixtures/04-all-inline-marks");

describe("Golden fixture: 04-all-inline-marks", () => {
  it("emits DOCX matching expected.document.xml", async () => {
    const envelope = JSON.parse(readFileSync(resolve(FIXTURE, "input.mddm.json"), "utf8")) as MDDMEnvelope;
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    const xml = await unzipDocxDocumentXml(blob);
    const actual = normalizeDocxXml(xml);

    const expectedPath = resolve(FIXTURE, "expected.document.xml");
    if (!existsSync(expectedPath)) {
      throw new Error(`Golden file missing: ${expectedPath}\nGenerate via MDDM_GOLDEN_UPDATE=1 plus the regenerator.`);
    }
    const expected = normalizeDocxXml(readFileSync(expectedPath, "utf8"));
    expect(actual).toBe(expected);
  });
});
