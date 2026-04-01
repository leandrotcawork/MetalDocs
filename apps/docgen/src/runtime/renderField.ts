import { Paragraph, Table, TableCell, TableRow, TextRun, WidthType } from "docx";
import { isRichBlockArray, renderRichBlocks, renderScalarValue } from "./blocks.js";
import type { FieldDef, RepeatFieldDef, RichFieldDef, TableFieldDef } from "./types.js";

function fieldLabel(field: { label: string }, depth: number): Paragraph {
  return new Paragraph({
    children: [new TextRun({ text: field.label, bold: true })],
    spacing: { before: 140, after: 60 },
    indent: depth > 0 ? { left: depth * 320 } : undefined,
  });
}

function renderScalarField(
  field: Extract<FieldDef, { type: "text" | "textarea" | "number" | "date" | "select" | "checkbox" }>,
  value: unknown,
  depth: number
): Paragraph[] {
  return [
    fieldLabel(field, depth),
    new Paragraph({
      children: [new TextRun({ text: renderScalarValue(value) })],
      indent: depth > 0 ? { left: depth * 320 } : undefined,
    }),
  ];
}

function renderTableField(field: TableFieldDef, value: unknown, depth: number): Array<Paragraph | Table> {
  const rows = Array.isArray(value) ? value : [];
  const header = new TableRow({
    children: field.columns.map(
      (column) =>
        new TableCell({
          children: [
            new Paragraph({
              children: [new TextRun({ text: column.label, bold: true })],
            }),
          ],
        })
    ),
  });

  const bodyRows = rows.map((row) => {
    const rowValue = typeof row === "object" && row !== null ? (row as Record<string, unknown>) : {};

    return new TableRow({
      children: field.columns.map(
        (column) =>
          new TableCell({
            children: [
              new Paragraph({
                children: [new TextRun({ text: renderScalarValue(rowValue[column.key]) })],
              }),
            ],
          })
      ),
    });
  });

  return [
    fieldLabel(field, depth),
    new Table({
      width: { size: 100, type: WidthType.PERCENTAGE },
      rows: [header, ...bodyRows],
    }),
  ];
}

function renderRichField(field: RichFieldDef, value: unknown, depth: number): Array<Paragraph | Table> {
  if (!isRichBlockArray(value)) {
    return [fieldLabel(field, depth), new Paragraph({ children: [new TextRun({ text: "N/A" })] })];
  }

  return [fieldLabel(field, depth), ...renderRichBlocks(value)];
}

function renderRepeatField(field: RepeatFieldDef, value: unknown, depth: number): Array<Paragraph | Table> {
  const items = Array.isArray(value) ? value : [];
  const nodes: Array<Paragraph | Table> = [fieldLabel(field, depth)];

  items.forEach((item, index) => {
    const record = typeof item === "object" && item !== null ? (item as Record<string, unknown>) : {};
    nodes.push(
      new Paragraph({
        children: [new TextRun({ text: `Item ${index + 1}`, bold: true })],
        spacing: { before: 120, after: 40 },
        indent: depth > 0 ? { left: depth * 320 } : undefined,
      })
    );
    nodes.push(...field.itemFields.flatMap((nestedField) => renderField(nestedField, record[nestedField.key], depth + 1)));
  });

  return nodes;
}

export function renderField(field: FieldDef, value: unknown, depth = 0): Array<Paragraph | Table> {
  switch (field.type) {
    case "text":
    case "textarea":
    case "number":
    case "date":
    case "select":
    case "checkbox":
      return renderScalarField(field, value, depth);
    case "table":
      return renderTableField(field, value, depth);
    case "rich":
      return renderRichField(field, value, depth);
    case "repeat":
      return renderRepeatField(field, value, depth);
  }
}
