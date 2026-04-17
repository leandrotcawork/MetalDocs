import { describe, it, expect } from "vitest"
import { htmlToExportTree } from "../html-to-export-tree"
import { emitDocxFromExportTree } from "../docx-emitter"
import { defaultLayoutTokens } from "../layout-ir"
import { readDocxDocumentXml } from "./helpers/read-docx-xml"

describe("hyperlink color sourced from LayoutTokens", () => {
  it("default tokens emit #0563C1 as a w:color attribute", async () => {
    const tree = htmlToExportTree('<p><a href="https://example.com">link</a></p>')
    const doc = emitDocxFromExportTree(tree, defaultLayoutTokens, new Map())
    const xml = await readDocxDocumentXml(doc)
    expect(xml).toMatch(/w:color[^>]*w:val="0563C1"/)
  })

  it("override token color appears in DOCX; old default does not", async () => {
    const tree = htmlToExportTree('<p><a href="https://example.com">link</a></p>')
    const override = {
      ...defaultLayoutTokens,
      theme: { ...defaultLayoutTokens.theme, hyperlink: "#FF0000" },
    }
    const doc = emitDocxFromExportTree(tree, override, new Map())
    const xml = await readDocxDocumentXml(doc)
    expect(xml).toMatch(/w:color[^>]*w:val="FF0000"/)
    expect(xml).not.toMatch(/w:color[^>]*w:val="0563C1"/)
  })
})

