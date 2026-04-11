import { describe, it } from "vitest";
import { readFileSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { mddmToDocx } from "../../docx-emitter";
import { defaultLayoutTokens } from "../../layout-ir";
import { normalizeDocxXml, unzipDocxDocumentXml } from "../golden-helpers";
import type { MDDMEnvelope } from "../../../adapter";

const FIXTURE_DIR = resolve(__dirname, "../fixtures/01-simple-po");

describe.skipIf(!process.env.MDDM_GOLDEN_UPDATE)("Golden regenerator (01-simple-po)", () => {
  it("writes expected.document.xml", async () => {
    const envelope = JSON.parse(readFileSync(resolve(FIXTURE_DIR, "input.mddm.json"), "utf8")) as MDDMEnvelope;
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    const xml = await unzipDocxDocumentXml(blob);
    const normalized = normalizeDocxXml(xml);
    writeFileSync(resolve(FIXTURE_DIR, "expected.document.xml"), normalized, "utf8");
  });
});
