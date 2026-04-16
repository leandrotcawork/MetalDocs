import { describe, expect, it } from "vitest"
import { htmlToExportTree } from "../html-to-export-tree"

describe("walker whitespace — inline containers keep spaces", () => {
  it("paragraph preserves whitespace between inline elements", () => {
    const tree = htmlToExportTree("<p>hello <b>bold</b> world</p>")
    expect(tree).toHaveLength(1)
    const p = tree[0]
    if (p.kind !== "paragraph") throw new Error("expected paragraph")
    const texts = p.children.filter((c) => c.kind === "text")
    expect(texts.length).toBeGreaterThanOrEqual(3)
    const values = texts.map((t) => (t.kind === "text" ? t.value : ""))
    expect(values).toContain("hello ")
    expect(values).toContain("bold")
    expect(values).toContain(" world")
  })

  it("heading preserves whitespace between inline elements", () => {
    const tree = htmlToExportTree("<h2>  Title  <em>note</em></h2>")
    expect(tree).toHaveLength(1)
    const h = tree[0]
    if (h.kind !== "heading") throw new Error("expected heading")
    const values = h.children
      .filter((c) => c.kind === "text")
      .map((t) => (t.kind === "text" ? t.value : ""))
    expect(values.some((v) => v.startsWith("  Title"))).toBe(true)
  })

  it("hyperlink keeps surrounding whitespace children", () => {
    const tree = htmlToExportTree('<p>see <a href="x">here</a> please</p>')
    const p = tree[0]
    if (p.kind !== "paragraph") throw new Error("expected paragraph")
    const values = p.children
      .filter((c) => c.kind === "text")
      .map((t) => (t.kind === "text" ? t.value : ""))
    expect(values).toContain("see ")
    expect(values).toContain(" please")
  })
})

describe("walker whitespace — block containers drop pure-whitespace text", () => {
  it("list item drops pure-whitespace text between block children", () => {
    const tree = htmlToExportTree("<ul><li>  <p>inner</p>  </li></ul>")
    const list = tree[0]
    if (list.kind !== "list") throw new Error("expected list")
    const li = list.items[0]
    const textRuns = li.children.filter((c) => c.kind === "text")
    expect(textRuns).toHaveLength(0)
  })

  it("list item keeps non-whitespace text directly inside", () => {
    const tree = htmlToExportTree("<ul><li>hello</li></ul>")
    const list = tree[0]
    if (list.kind !== "list") throw new Error("expected list")
    const li = list.items[0]
    const textRuns = li.children.filter((c) => c.kind === "text")
    expect(textRuns).toHaveLength(1)
    if (textRuns[0].kind === "text") expect(textRuns[0].value).toBe("hello")
  })

  it("blockquote drops pure-whitespace between nested paragraphs", () => {
    const tree = htmlToExportTree("<blockquote>  <p>one</p>  <p>two</p>  </blockquote>")
    const bq = tree[0]
    if (bq.kind !== "blockquote") throw new Error("expected blockquote")
    const paragraphs = bq.children.filter((c) => c.kind === "paragraph")
    const textBetween = bq.children.filter((c) => c.kind === "text")
    expect(paragraphs).toHaveLength(2)
    expect(textBetween).toHaveLength(0)
  })

  it("table cell drops pure-whitespace text between block children", () => {
    const html =
      '<figure class="table"><table><tr><td>  <p>cell</p>  </td></tr></table></figure>'
    const tree = htmlToExportTree(html)
    const table = tree[0]
    if (table.kind !== "table") throw new Error("expected table")
    const cell = table.rows[0].cells[0]
    const texts = cell.children.filter((c) => c.kind === "text")
    expect(texts).toHaveLength(0)
    const paragraphs = cell.children.filter((c) => c.kind === "paragraph")
    expect(paragraphs).toHaveLength(1)
  })

  it("repeatable item drops pure-whitespace between nested blocks", () => {
    const html =
      '<ol class="mddm-repeatable"><li>  <p>one</p>  </li></ol>'
    const tree = htmlToExportTree(html)
    const rep = tree[0]
    if (rep.kind !== "repeatable") throw new Error("expected repeatable")
    const item = rep.items[0]
    const texts = item.children.filter((c) => c.kind === "text")
    expect(texts).toHaveLength(0)
  })

  it("section body drops pure-whitespace between nested blocks", () => {
    const html =
      '<section class="mddm-section">' +
      '<div class="mddm-section-body">  <p>body</p>  </div>' +
      "</section>"
    const tree = htmlToExportTree(html)
    const s = tree[0]
    if (s.kind !== "section") throw new Error("expected section")
    const texts = s.body.filter((c) => c.kind === "text")
    expect(texts).toHaveLength(0)
  })
})

describe("walker whitespace — context-preserving wrappers", () => {
  it("SPAN.restricted-editing-exception preserves whitespace inside a paragraph (inline context)", () => {
    const tree = htmlToExportTree(
      '<p>before <span class="restricted-editing-exception">mid </span>after</p>',
    )
    const p = tree[0]
    if (p.kind !== "paragraph") throw new Error("expected paragraph")
    const values = p.children
      .filter((c) => c.kind === "text")
      .map((t) => (t.kind === "text" ? t.value : ""))
    expect(values).toContain("before ")
    expect(values).toContain("mid ")
    expect(values).toContain("after")
  })

  it("DIV.mddm-rich-block drops pure-whitespace between nested blocks (block context)", () => {
    const tree = htmlToExportTree(
      '<blockquote>  <div class="mddm-rich-block">  <p>one</p>  <p>two</p>  </div>  </blockquote>',
    )
    const bq = tree[0]
    if (bq.kind !== "blockquote") throw new Error("expected blockquote")
    const paragraphs = bq.children.filter((c) => c.kind === "paragraph")
    const textRuns = bq.children.filter((c) => c.kind === "text")
    expect(paragraphs).toHaveLength(2)
    expect(textRuns).toHaveLength(0)
  })
})
