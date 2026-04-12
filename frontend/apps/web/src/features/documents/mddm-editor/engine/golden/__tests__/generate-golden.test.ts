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

describe.skipIf(!process.env.MDDM_GOLDEN_UPDATE)("Golden regenerator (02-05 + 06 fixtures)", () => {
  it("writes expected.document.xml for 02-complex-table", async () => {
    const dir = resolve(__dirname, "../fixtures/02-complex-table");
    const envelope = JSON.parse(readFileSync(resolve(dir, "input.mddm.json"), "utf8")) as MDDMEnvelope;
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    const xml = await unzipDocxDocumentXml(blob);
    writeFileSync(resolve(dir, "expected.document.xml"), xml, "utf8");
  });

  it("writes expected.document.xml for 03-repeatable-sections", async () => {
    const dir = resolve(__dirname, "../fixtures/03-repeatable-sections");
    const envelope = JSON.parse(readFileSync(resolve(dir, "input.mddm.json"), "utf8")) as MDDMEnvelope;
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    const xml = await unzipDocxDocumentXml(blob);
    writeFileSync(resolve(dir, "expected.document.xml"), xml, "utf8");
  });

  it("writes expected.document.xml for 04-all-inline-marks", async () => {
    const dir = resolve(__dirname, "../fixtures/04-all-inline-marks");
    const envelope = JSON.parse(readFileSync(resolve(dir, "input.mddm.json"), "utf8")) as MDDMEnvelope;
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    const xml = await unzipDocxDocumentXml(blob);
    writeFileSync(resolve(dir, "expected.document.xml"), xml, "utf8");
  });

  it("writes expected.document.xml for 05-multi-block-doc", async () => {
    const dir = resolve(__dirname, "../fixtures/05-multi-block-doc");
    const envelope = JSON.parse(readFileSync(resolve(dir, "input.mddm.json"), "utf8")) as MDDMEnvelope;
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    const xml = await unzipDocxDocumentXml(blob);
    writeFileSync(resolve(dir, "expected.document.xml"), xml, "utf8");
  });

  it("writes expected.document.xml for 06-theme-override", async () => {
    const dir = resolve(__dirname, "../fixtures/06-theme-override");
    const envelope = JSON.parse(readFileSync(resolve(dir, "input.mddm.json"), "utf8")) as MDDMEnvelope;
    const tokens = {
      ...defaultLayoutTokens,
      theme: { accent: "#2a4f8b", accentLight: "#eaf1fa", accentDark: "#15273f", accentBorder: "#b9c9e0" },
    };
    const blob = await mddmToDocx(envelope, tokens);
    const xml = await unzipDocxDocumentXml(blob);
    writeFileSync(resolve(dir, "expected.document.xml"), xml, "utf8");
  });

  it("writes expected.document.xml for 07-repeatable-nested", async () => {
    const dir = resolve(__dirname, "../fixtures/07-repeatable-nested");
    const envelope = JSON.parse(readFileSync(resolve(dir, "input.mddm.json"), "utf8")) as MDDMEnvelope;
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    const xml = await unzipDocxDocumentXml(blob);
    writeFileSync(resolve(dir, "expected.document.xml"), xml, "utf8");
  });
});
