import { readFileSync } from "node:fs"
import { fileURLToPath } from "node:url"
import { describe, expect, it } from "vitest"
import type { ExportNode } from "../export-node"
import { htmlToExportTree } from "../html-to-export-tree"

function readFixture(name: string): string {
  return readFileSync(fileURLToPath(new URL(`../__fixtures__/${name}`, import.meta.url)), "utf8")
}

describe("htmlToExportTree", () => {
  it("parses section-with-fields.html", () => {
    const html = readFixture("section-with-fields.html")
    const out = htmlToExportTree(html)

    const expected: ExportNode[] = [
      {
        kind: "section",
        variant: "bordered",
        header: [{ kind: "heading", level: 2, children: [{ kind: "text", value: "Client Info" }] }],
        body: [
          {
            kind: "paragraph",
            children: [
              { kind: "text", value: "Name: " },
              { kind: "field", id: "client_name", fieldType: "text", value: "ACME Corp" },
            ],
          },
          {
            kind: "paragraph",
            children: [
              { kind: "text", value: "Date: " },
              { kind: "field", id: "order_date", fieldType: "date", value: "2026-04-16" },
            ],
          },
        ],
      },
    ]

    expect(out).toEqual(expected)
  })

  it("parses table-fixed.html", () => {
    const html = readFixture("table-fixed.html")
    const out = htmlToExportTree(html)

    const expected: ExportNode[] = [
      {
        kind: "table",
        variant: "fixed",
        rows: [
          {
            kind: "tableRow",
            cells: [
              { kind: "tableCell", isHeader: true, children: [{ kind: "text", value: "Item" }] },
              { kind: "tableCell", isHeader: true, children: [{ kind: "text", value: "Qty" }] },
            ],
          },
          {
            kind: "tableRow",
            cells: [
              { kind: "tableCell", isHeader: false, children: [{ kind: "text", value: "Widget" }] },
              { kind: "tableCell", isHeader: false, children: [{ kind: "text", value: "10" }] },
            ],
          },
        ],
      },
    ]

    expect(out).toEqual(expected)
  })

  it("parses table-dynamic.html", () => {
    const html = readFixture("table-dynamic.html")
    const out = htmlToExportTree(html)

    const expected: ExportNode[] = [
      {
        kind: "table",
        variant: "dynamic",
        rows: [
          {
            kind: "tableRow",
            cells: [
              { kind: "tableCell", isHeader: true, children: [{ kind: "text", value: "Item" }] },
              { kind: "tableCell", isHeader: true, children: [{ kind: "text", value: "Qty" }] },
            ],
          },
          {
            kind: "tableRow",
            cells: [
              { kind: "tableCell", isHeader: false, children: [{ kind: "text", value: "Widget" }] },
              { kind: "tableCell", isHeader: false, children: [{ kind: "text", value: "10" }] },
            ],
          },
        ],
      },
    ]

    expect(out).toEqual(expected)
  })

  it("parses repeatable.html", () => {
    const html = readFixture("repeatable.html")
    const out = htmlToExportTree(html)

    const expected: ExportNode[] = [
      {
        kind: "repeatable",
        items: [
          { kind: "repeatableItem", children: [{ kind: "paragraph", children: [{ kind: "text", value: "Row one" }] }] },
          { kind: "repeatableItem", children: [{ kind: "paragraph", children: [{ kind: "text", value: "Row two" }] }] },
        ],
      },
    ]

    expect(out).toEqual(expected)
  })

  it("parses rich-block.html", () => {
    const html = readFixture("rich-block.html")
    const out = htmlToExportTree(html)

    const expected: ExportNode[] = [
      { kind: "paragraph", children: [{ kind: "text", value: "Wrapped paragraph should be unwrapped." }] },
      {
        kind: "list",
        ordered: false,
        items: [
          { kind: "listItem", children: [{ kind: "text", value: "One" }] },
          { kind: "listItem", children: [{ kind: "text", value: "Two" }] },
        ],
      },
    ]

    expect(out).toEqual(expected)
  })

  it("parses nested-formatting.html", () => {
    const html = readFixture("nested-formatting.html")
    const out = htmlToExportTree(html)

    const expected: ExportNode[] = [
      {
        kind: "paragraph",
        children: [
          { kind: "text", value: "Hello " },
          { kind: "text", value: "bold ", marks: ["bold"] },
          { kind: "text", value: "italic", marks: ["bold", "italic"] },
          { kind: "text", value: " and " },
          { kind: "hyperlink", href: "https://example.com", children: [{ kind: "text", value: "link" }] },
          { kind: "text", value: "." },
          { kind: "lineBreak" },
          { kind: "text", value: "Line 2." },
        ],
      },
    ]

    expect(out).toEqual(expected)
  })
})
