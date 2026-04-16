// Discriminated union for CK5 HTML export tree

export interface Section {
  kind: "section"
  variant: "solid" | "bordered" | "plain"
  header?: ExportNode[]
  body: ExportNode[]
}

export interface Repeatable {
  kind: "repeatable"
  items: RepeatableItem[]
}

export interface RepeatableItem {
  kind: "repeatableItem"
  children: ExportNode[]
}

export interface Table {
  kind: "table"
  variant: "fixed" | "dynamic"
  rows: TableRow[]
}

export interface TableRow {
  kind: "tableRow"
  cells: TableCell[]
}

export interface TableCell {
  kind: "tableCell"
  isHeader: boolean
  children: ExportNode[]
  colspan?: number
  rowspan?: number
}

export interface Field {
  kind: "field"
  id: string
  fieldType: "text" | "number" | "date" | "boolean" | "select"
  value: string
}

export interface Heading {
  kind: "heading"
  level: 1 | 2 | 3 | 4 | 5 | 6
  children: ExportNode[]
}

export interface Paragraph {
  kind: "paragraph"
  align?: "left" | "right" | "center" | "justify"
  children: ExportNode[]
}

export interface List {
  kind: "list"
  ordered: boolean
  items: ListItem[]
}

export interface ListItem {
  kind: "listItem"
  children: ExportNode[]
}

export interface Image {
  kind: "image"
  src: string
  alt?: string
  width?: number
  height?: number
}

export interface Hyperlink {
  kind: "hyperlink"
  href: string
  children: ExportNode[]
}

export interface Text {
  kind: "text"
  value: string
  marks?: ("bold" | "italic" | "underline" | "strike")[]
}

export interface LineBreak {
  kind: "lineBreak"
}

export interface Blockquote {
  kind: "blockquote"
  children: ExportNode[]
}

export type ExportNode =
  | Section
  | Repeatable
  | RepeatableItem
  | Table
  | TableRow
  | TableCell
  | Field
  | Heading
  | Paragraph
  | List
  | ListItem
  | Image
  | Hyperlink
  | Text
  | LineBreak
  | Blockquote
