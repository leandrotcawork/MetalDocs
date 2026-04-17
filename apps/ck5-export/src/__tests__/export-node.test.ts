import { describe, it } from "vitest"
import type { ExportNode, Heading, Paragraph, Section, Text } from "../export-node.js"

describe("ExportNode types", () => {
  it("literal section satisfies ExportNode", () => {
    const node = {
      kind: "section" as const,
      variant: "bordered" as const,
      body: [
        {
          kind: "paragraph" as const,
          children: [{ kind: "text" as const, value: "Hello" }],
        } satisfies Paragraph,
      ],
    } satisfies Section

    const asExportNode: ExportNode = node
    void asExportNode
  })

  it("literal text satisfies ExportNode", () => {
    const node = {
      kind: "text" as const,
      value: "hello",
      marks: ["bold" as const, "italic" as const],
    } satisfies Text

    const asExportNode: ExportNode = node
    void asExportNode
  })

  it("heading with children satisfies ExportNode", () => {
    const node = {
      kind: "heading" as const,
      level: 1 as const,
      children: [],
    } satisfies Heading

    const asExportNode: ExportNode = node
    void asExportNode
  })
})
