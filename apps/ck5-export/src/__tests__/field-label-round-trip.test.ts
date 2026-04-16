import { describe, it, expect } from "vitest"
import { htmlToExportTree } from "../html-to-export-tree"
import { emitDocxFromExportTree } from "../ck5-docx-emitter"
import { defaultLayoutTokens } from "../layout-ir"
import { readDocxDocumentXml } from "./helpers/read-docx-xml"

describe("field label round-trip", () => {
  const html =
    '<span class="mddm-field" data-field-id="customer" data-field-type="text" data-field-label="Customer">Acme</span>'

  it("walker captures label from data-field-label", () => {
    const tree = htmlToExportTree(html)
    expect(tree).toHaveLength(1)
    const node = tree[0]
    expect(node.kind).toBe("field")
    if (node.kind !== "field") return
    expect(node.id).toBe("customer")
    expect(node.label).toBe("Customer")
    expect(node.value).toBe("Acme")
  })

  it("emitter prints label in DOCX label cell", async () => {
    const tree = htmlToExportTree(html)
    const doc = emitDocxFromExportTree(tree, defaultLayoutTokens, new Map())
    const xml = await readDocxDocumentXml(doc)
    expect(xml).toMatch(/<w:t[^>]*>Customer<\/w:t>/)
    expect(xml).not.toMatch(/<w:t[^>]*>customer<\/w:t>/)
  })
})
