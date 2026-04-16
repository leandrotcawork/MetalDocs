import { parseHTML } from "linkedom"
import type { ExportNode, Field, Paragraph, TableCell, TableRow, Text } from "./export-node"

type TextMark = NonNullable<Text["marks"]>[number]

const ELEMENT_NODE = 1
const TEXT_NODE = 3

export function htmlToExportTree(html: string): ExportNode[] {
  const { document } = parseHTML(`<!DOCTYPE html><html><body>${html}</body></html>`)
  return walkChildren(document.body, false, [])
}

function walkChildren(parent: ParentNode, inlineContext: boolean, marks: TextMark[]): ExportNode[] {
  const out: ExportNode[] = []

  for (const child of Array.from(parent.childNodes)) {
    out.push(...walkNode(child, inlineContext, marks))
  }

  return out
}

function walkNode(node: Node, inlineContext: boolean, marks: TextMark[]): ExportNode[] {
  if (node.nodeType === TEXT_NODE) {
    const value = node.textContent ?? ""
    if (!inlineContext && isWhitespaceOnly(value)) {
      return []
    }

    return [buildText(value, marks)]
  }

  if (node.nodeType !== ELEMENT_NODE) {
    return []
  }

  const el = node as HTMLElement
  const tagName = el.tagName.toUpperCase()

  if (tagName === "DIV" && el.classList.contains("mddm-rich-block")) {
    return walkChildren(el, inlineContext, marks)
  }

  if (tagName === "SPAN" && el.classList.contains("restricted-editing-exception")) {
    return walkChildren(el, inlineContext, marks)
  }

  if (tagName === "STRONG" || tagName === "B") {
    return walkChildren(el, true, addMark(marks, "bold"))
  }

  if (tagName === "EM" || tagName === "I") {
    return walkChildren(el, true, addMark(marks, "italic"))
  }

  if (tagName === "U") {
    return walkChildren(el, true, addMark(marks, "underline"))
  }

  if (tagName === "S" || tagName === "STRIKE") {
    return walkChildren(el, true, addMark(marks, "strike"))
  }

  if (tagName === "SECTION" && el.classList.contains("mddm-section")) {
    const headerContainer = findDirectChildByClass(el, "mddm-section-header")
    const bodyContainer = findDirectChildByClass(el, "mddm-section-body")

    const header = headerContainer ? walkChildren(headerContainer, false, []) : undefined
    const body = bodyContainer ? walkChildren(bodyContainer, false, []) : []

    return [
      {
        kind: "section",
        variant: (el.getAttribute("data-variant") || "plain") as "solid" | "bordered" | "plain",
        header: header && header.length > 0 ? header : undefined,
        body,
      },
    ]
  }

  if (tagName === "OL" && el.classList.contains("mddm-repeatable")) {
    const items = Array.from(el.children)
      .filter((child) => child.tagName.toUpperCase() === "LI")
      .map((li) => ({
        kind: "repeatableItem" as const,
        children: walkChildren(li, false, []),
      }))

    return [
      {
        kind: "repeatable",
        items,
      },
    ]
  }

  if (tagName === "FIGURE" && el.classList.contains("table")) {
    const tableEl = el.querySelector("table")
    const rows = tableEl ? collectTableRows(tableEl) : []

    return [
      {
        kind: "table",
        variant: (el.getAttribute("data-variant") || "fixed") as "fixed" | "dynamic",
        rows,
      },
    ]
  }

  if (tagName === "SPAN" && el.classList.contains("mddm-field")) {
    const label = el.getAttribute("data-field-label") ?? undefined
    return [
      {
        kind: "field",
        id: el.getAttribute("data-field-id") ?? "",
        label: label && label.length > 0 ? label : undefined,
        fieldType: (el.getAttribute("data-field-type") ?? "text") as Field["fieldType"],
        value: el.textContent ?? "",
      },
    ]
  }

  if (/^H[1-6]$/.test(tagName)) {
    return [
      {
        kind: "heading",
        level: Number.parseInt(tagName[1] ?? "1", 10) as 1 | 2 | 3 | 4 | 5 | 6,
        children: walkChildren(el, true, marks),
      },
    ]
  }

  if (tagName === "P") {
    const align = (el.style.textAlign || undefined) as Paragraph["align"]

    return [
      {
        kind: "paragraph",
        align,
        children: walkChildren(el, true, marks),
      },
    ]
  }

  if (tagName === "UL" || (tagName === "OL" && !el.classList.contains("mddm-repeatable"))) {
    const items = Array.from(el.children)
      .filter((child) => child.tagName.toUpperCase() === "LI")
      .map((li) => ({
        kind: "listItem" as const,
        children: walkChildren(li, false, marks),
      }))

    return [
      {
        kind: "list",
        ordered: tagName === "OL",
        items,
      },
    ]
  }

  // LI nodes are collected directly by the UL/OL branch above via Array.from(el.children).
  // A bare <li> outside a list is not valid CK5 HTML; fall through to walkChildren.

  if (tagName === "IMG") {
    const imageEl = el as HTMLImageElement
    const width = imageEl.width || parseOptionalNumber(el.getAttribute("width"))
    const height = imageEl.height || parseOptionalNumber(el.getAttribute("height"))

    return [
      {
        kind: "image",
        src: el.getAttribute("src") ?? "",
        alt: el.getAttribute("alt") || undefined,
        width,
        height,
      },
    ]
  }

  if (tagName === "A") {
    return [
      {
        kind: "hyperlink",
        href: (el as HTMLAnchorElement).href,
        children: walkChildren(el, true, marks),
      },
    ]
  }

  if (tagName === "BR") {
    return [{ kind: "lineBreak" }]
  }

  if (tagName === "BLOCKQUOTE") {
    return [
      {
        kind: "blockquote",
        children: walkChildren(el, false, marks),
      },
    ]
  }

  // TR/TH/TD nodes are handled exclusively by collectTableRows → collectCells → buildTableCell.
  // They are never reached through walkNode in valid CK5 HTML.

  return walkChildren(el, inlineContext, marks)
}

function collectTableRows(tableEl: Element): TableRow[] {
  const rows: TableRow[] = []
  for (const row of Array.from(tableEl.querySelectorAll("tr"))) {
    rows.push({
      kind: "tableRow",
      cells: collectCells(row),
    })
  }
  return rows
}

function collectCells(rowEl: Element): TableCell[] {
  const cells: TableCell[] = []

  for (const child of Array.from(rowEl.children)) {
    const tagName = child.tagName.toUpperCase()
    if (tagName === "TH" || tagName === "TD") {
      cells.push(buildTableCell(child))
    }
  }

  return cells
}

function buildTableCell(el: Element): TableCell {
  const colspan = parseOptionalNumber(el.getAttribute("colspan"))
  const rowspan = parseOptionalNumber(el.getAttribute("rowspan"))

  return {
    kind: "tableCell",
    isHeader: el.tagName.toUpperCase() === "TH",
    children: walkChildren(el, false, []),
    colspan,
    rowspan,
  }
}

function buildText(value: string, marks: TextMark[]): Text {
  if (marks.length === 0) {
    return { kind: "text", value }
  }

  return {
    kind: "text",
    value,
    marks: [...marks],
  }
}

function addMark(current: TextMark[], mark: TextMark): TextMark[] {
  return current.includes(mark) ? current : [...current, mark]
}

function isWhitespaceOnly(value: string): boolean {
  return value.trim().length === 0
}

function parseOptionalNumber(value: string | null): number | undefined {
  if (!value) {
    return undefined
  }

  const parsed = Number.parseInt(value, 10)
  return Number.isFinite(parsed) ? parsed : undefined
}

function findDirectChildByClass(el: Element, className: string): HTMLElement | undefined {
  for (const child of Array.from(el.children)) {
    if (child.classList.contains(className)) {
      return child as HTMLElement
    }
  }

  return undefined
}
