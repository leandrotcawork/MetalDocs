import { describe, it, expect } from "vitest"
import { htmlToExportTree } from "../html-to-export-tree"
import { collectImageUrls, emitDocxFromExportTree } from "../docx-emitter"
import { defaultLayoutTokens } from "../layout-tokens"
import type { ResolvedAsset } from "../asset-resolver"
import { readDocxDocumentXml } from "./helpers/read-docx-xml"

const emptyAssets = new Map<string, ResolvedAsset>()

async function emit(html: string, assets = emptyAssets): Promise<string> {
  const tree = htmlToExportTree(html)
  const doc = emitDocxFromExportTree(tree, defaultLayoutTokens, assets)
  return readDocxDocumentXml(doc)
}

describe("emitter — paragraph", () => {
  it("renders a plain paragraph with text run", async () => {
    const xml = await emit("<p>Hello world</p>")
    expect(xml).toMatch(/<w:p[\s>]/)
    expect(xml).toMatch(/<w:t[^>]*>Hello world<\/w:t>/)
  })

  it("renders bold + italic + underline marks", async () => {
    const xml = await emit("<p><strong>b</strong><em>i</em><u>u</u></p>")
    expect(xml).toMatch(/<w:b[\s/]/)
    expect(xml).toMatch(/<w:i[\s/]/)
    expect(xml).toMatch(/<w:u[\s/]/)
  })
})

describe("emitter — heading", () => {
  it("emits heading style for H1..H3", async () => {
    const xml = await emit("<h1>one</h1><h2>two</h2><h3>three</h3>")
    expect(xml).toMatch(/w:pStyle[^>]*w:val="Heading1"/)
    expect(xml).toMatch(/w:pStyle[^>]*w:val="Heading2"/)
    expect(xml).toMatch(/w:pStyle[^>]*w:val="Heading3"/)
  })
})

describe("emitter — list", () => {
  it("bulleted list emits numPr/ilvl", async () => {
    const xml = await emit("<ul><li>a</li><li>b</li></ul>")
    expect(xml).toMatch(/<w:numPr>/)
    expect(xml).toMatch(/<w:t[^>]*>a<\/w:t>/)
    expect(xml).toMatch(/<w:t[^>]*>b<\/w:t>/)
  })

  it("ordered list emits numPr with numbering reference", async () => {
    const xml = await emit("<ol><li>a</li></ol>")
    expect(xml).toMatch(/<w:numPr>/)
  })
})

describe("emitter — blockquote", () => {
  it("blockquote renders with left indent", async () => {
    const xml = await emit("<blockquote>quote</blockquote>")
    expect(xml).toMatch(/<w:ind[^>]*w:left="720"/)
    expect(xml).toMatch(/<w:t[^>]*>quote<\/w:t>/)
  })
})

describe("emitter — image", () => {
  it("skips images whose asset was not resolved", async () => {
    const xml = await emit('<img src="/assets/missing.png">')
    expect(xml).not.toMatch(/<w:drawing/)
  })

  it("renders a drawing when the asset resolves", async () => {
    const bytes = Buffer.from(
      // 1x1 transparent PNG
      "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=",
      "base64",
    )
    const assets = new Map<string, ResolvedAsset>([
      ["/assets/one.png", { bytes, mimeType: "image/png" }],
    ])
    const xml = await emit('<img src="/assets/one.png" width="10" height="10">', assets)
    expect(xml).toMatch(/<w:drawing/)
  })
})

describe("emitter — field", () => {
  it("renders a 2-column table with label and value", async () => {
    const xml = await emit(
      '<span class="mddm-field" data-field-id="x" data-field-label="X" data-field-type="text">v</span>',
    )
    expect(xml).toMatch(/<w:tbl[\s>]/)
    expect(xml).toMatch(/<w:t[^>]*>X<\/w:t>/)
    expect(xml).toMatch(/<w:t[^>]*>v<\/w:t>/)
  })
})

describe("emitter — hyperlink", () => {
  it("renders a hyperlink text run", async () => {
    const xml = await emit('<p><a href="https://example.com">link</a></p>')
    expect(xml).toMatch(/<w:t[^>]*>link<\/w:t>/)
    expect(xml).toMatch(/w:color[^>]*w:val="0563C1"/)
  })
})

describe("emitter — table", () => {
  it("renders figure.table → w:tbl with rows and cells", async () => {
    const xml = await emit(
      '<figure class="table"><table><tbody>' +
        "<tr><th>h1</th><th>h2</th></tr>" +
        "<tr><td>a</td><td>b</td></tr>" +
        "</tbody></table></figure>",
    )
    expect(xml).toMatch(/<w:tbl[\s>]/)
    expect(xml).toMatch(/<w:t[^>]*>h1<\/w:t>/)
    expect(xml).toMatch(/<w:t[^>]*>h2<\/w:t>/)
    expect(xml).toMatch(/<w:t[^>]*>a<\/w:t>/)
    expect(xml).toMatch(/<w:t[^>]*>b<\/w:t>/)
  })
})

describe("emitter — section", () => {
  it("section with header + body emits two rows in a framing table", async () => {
    const xml = await emit(
      '<section class="mddm-section" data-variant="bordered">' +
        '<header class="mddm-section-header"><h2>Title</h2></header>' +
        '<div class="mddm-section-body"><p>body</p></div>' +
        "</section>",
    )
    expect(xml).toMatch(/<w:t[^>]*>Title<\/w:t>/)
    expect(xml).toMatch(/<w:t[^>]*>body<\/w:t>/)
    // Two rows (header + body) in the section's framing table.
    const rows = xml.match(/<w:tr[\s>]/g) ?? []
    expect(rows.length).toBeGreaterThanOrEqual(2)
  })
})

describe("emitter — repeatable", () => {
  it("each repeatable item emits its own framing table", async () => {
    const xml = await emit(
      '<ol class="mddm-repeatable">' + "<li><p>one</p></li><li><p>two</p></li>" + "</ol>",
    )
    expect(xml).toMatch(/<w:t[^>]*>one<\/w:t>/)
    expect(xml).toMatch(/<w:t[^>]*>two<\/w:t>/)
    // At least two tables (one per item).
    const tables = xml.match(/<w:tbl[\s>]/g) ?? []
    expect(tables.length).toBeGreaterThanOrEqual(2)
  })
})

describe("asset-collector — collectImageUrls", () => {
  it("collects urls from nested sections, repeatables, lists, tables", () => {
    const html =
      '<section class="mddm-section">' +
      '<div class="mddm-section-body">' +
      '<p><img src="/a.png"></p>' +
      '<ol class="mddm-repeatable"><li><p><img src="/b.png"></p></li></ol>' +
      "<ul><li><img src=\"/c.png\"></li></ul>" +
      '<figure class="table"><table><tr><td><img src="/d.png"></td></tr></table></figure>' +
      "</div></section>"
    const tree = htmlToExportTree(html)
    const urls = collectImageUrls(tree)
    expect(urls.sort()).toEqual(["/a.png", "/b.png", "/c.png", "/d.png"])
  })

  it("deduplicates repeated urls", () => {
    const html = '<p><img src="/same.png"></p><p><img src="/same.png"></p>'
    const tree = htmlToExportTree(html)
    expect(collectImageUrls(tree)).toEqual(["/same.png"])
  })
})
