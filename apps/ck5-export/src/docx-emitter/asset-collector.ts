import type { ExportNode } from "../export-node"

export function collectImageUrls(nodes: ExportNode[]): string[] {
  const urls = new Set<string>()

  const walk = (items: ExportNode[]) => {
    for (const node of items) {
      switch (node.kind) {
        case "image":
          if (node.src.length > 0) {
            urls.add(node.src)
          }
          break
        case "section":
          if (node.header) {
            walk(node.header)
          }
          walk(node.body)
          break
        case "repeatable":
          for (const item of node.items) {
            walk(item.children)
          }
          break
        case "repeatableItem":
        case "paragraph":
        case "heading":
        case "blockquote":
        case "hyperlink":
        case "listItem":
        case "tableCell":
          walk(node.children)
          break
        case "list":
          for (const item of node.items) {
            walk(item.children)
          }
          break
        case "table":
          for (const row of node.rows) {
            for (const cell of row.cells) {
              walk(cell.children)
            }
          }
          break
        case "tableRow":
          for (const cell of node.cells) {
            walk(cell.children)
          }
          break
        case "field":
        case "text":
        case "lineBreak":
          break
      }
    }
  }

  walk(nodes)
  return [...urls]
}
