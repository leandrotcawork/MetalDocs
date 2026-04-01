import { BorderStyle, Paragraph, Table, TableRow } from "docx";
import { renderRichBlocks, renderScalarValue } from "./blocks.js";
import {
  C,
  CONTENT_WIDTH,
  LABEL_VALUE_ROW,
  REPEAT_ROW,
  fieldParagraph,
  makeCell,
  makeTable,
  mixWithWhite,
  tableBorder,
} from "./docx.js";
import type { FieldDef, RepeatFieldDef, RichFieldDef, ScalarFieldType, TableFieldDef } from "./types.js";

type RenderContext = {
  sectionColor?: string;
};

const scalarTypes: ScalarFieldType[] = ["text", "textarea", "number", "date", "select", "checkbox"];

function isScalarField(field: FieldDef): field is Extract<FieldDef, { type: ScalarFieldType }> {
  return scalarTypes.includes(field.type as ScalarFieldType);
}

function renderScalarTable(field: Extract<FieldDef, { type: ScalarFieldType }>, value: unknown): Table {
  const labelCell = makeCell({
    width: LABEL_VALUE_ROW[0],
    fill: C.grayLight,
    children: [
      fieldParagraph(field.label, {
        bold: true,
        color: C.gray,
        size: 18,
      }),
    ],
  });

  const valueCell = makeCell({
    width: LABEL_VALUE_ROW[1],
    children: [
      fieldParagraph(renderScalarValue(value), {
        italic: true,
        color: "2C2C2A",
        size: 18,
      }),
    ],
  });

  return makeTable([new TableRow({ children: [labelCell, valueCell] })], LABEL_VALUE_ROW, {
    width: CONTENT_WIDTH,
    borders: tableBorder(BorderStyle.NONE),
  });
}

function renderTableField(field: TableFieldDef, value: unknown): Array<Paragraph | Table> {
  const rows = Array.isArray(value) ? value : [];
  const columns = field.columns.length;
  const columnWidth = Math.floor(CONTENT_WIDTH / columns);
  const columnWidths = Array.from({ length: columns }, () => columnWidth);

  const header = new TableRow({
    children: field.columns.map((column) =>
      makeCell({
        width: columnWidth,
        fill: C.grayLight,
        children: [
          fieldParagraph(column.label, {
            bold: true,
            color: C.gray,
            size: 18,
          }),
        ],
      })
    ),
  });

  const bodyRows = rows.map((row) => {
    const rowValue = typeof row === "object" && row !== null ? (row as Record<string, unknown>) : {};
    return new TableRow({
      children: field.columns.map((column) =>
        makeCell({
          width: columnWidth,
          children: [
            fieldParagraph(renderScalarValue(rowValue[column.key]), {
              italic: true,
              color: "2C2C2A",
              size: 18,
            }),
          ],
        })
      ),
    });
  });

  return [
    makeTable([header, ...bodyRows], columnWidths, {
      width: CONTENT_WIDTH,
      borders: tableBorder(BorderStyle.NONE),
    }),
  ];
}

function renderRepeatScalarGrid(fields: readonly FieldDef[], record: Record<string, unknown>): Array<Paragraph | Table> {
  const scalarFields = fields.filter(isScalarField);
  const nodes: Array<Paragraph | Table> = [];

  for (let index = 0; index < scalarFields.length; index += 2) {
    const left = scalarFields[index];
    const right = scalarFields[index + 1];
    if (!right) {
      nodes.push(
        makeTable(
          [
            new TableRow({
              children: [
                makeCell({
                  width: CONTENT_WIDTH,
                  fill: C.grayLight,
                  children: [
                    fieldParagraph(left.label, {
                      bold: true,
                      color: C.gray,
                      size: 18,
                    }),
                    fieldParagraph(renderScalarValue(record[left.key]), {
                      italic: true,
                      color: "2C2C2A",
                      size: 18,
                    }),
                  ],
                }),
              ],
            }),
          ],
          [CONTENT_WIDTH],
          { width: CONTENT_WIDTH, borders: tableBorder(BorderStyle.NONE) }
        )
      );
      continue;
    }

    const pairTable = makeTable(
      [
        new TableRow({
          children: [
            makeCell({
              width: REPEAT_ROW[0],
              fill: C.grayLight,
              children: [
                fieldParagraph(left.label, { bold: true, color: C.gray, size: 18 }),
                fieldParagraph(renderScalarValue(record[left.key]), {
                  italic: true,
                  color: "2C2C2A",
                  size: 18,
                }),
              ],
            }),
            makeCell({
              width: REPEAT_ROW[1],
              fill: C.grayLight,
              children: [
                fieldParagraph(right.label, { bold: true, color: C.gray, size: 18 }),
                fieldParagraph(renderScalarValue(record[right.key]), {
                  italic: true,
                  color: "2C2C2A",
                  size: 18,
                }),
              ],
            }),
          ],
        }),
      ],
      REPEAT_ROW,
      { width: CONTENT_WIDTH, borders: tableBorder(BorderStyle.NONE) }
    );
    nodes.push(pairTable);
  }

  return nodes;
}

function renderRepeatField(field: RepeatFieldDef, value: unknown, context: RenderContext): Array<Paragraph | Table> {
  const items = Array.isArray(value) ? value : [];
  const nodes: Array<Paragraph | Table> = [];
  const sectionColor = (context.sectionColor ?? C.gray).replace(/^#/, "").toUpperCase();
  const headerTextColor = sectionColor === C.coral ? C.coral : sectionColor;
  const headerBold = sectionColor === C.coral;

  items.forEach((item, index) => {
    const record = typeof item === "object" && item !== null ? (item as Record<string, unknown>) : {};
    nodes.push(
      makeTable(
        [
          new TableRow({
            children: [
              makeCell({
                width: CONTENT_WIDTH,
                fill: mixWithWhite(sectionColor, 0.3),
                children: [
                  fieldParagraph(`Item ${index + 1}`, {
                    bold: headerBold,
                    color: headerTextColor,
                    size: 18,
                  }),
                ],
              }),
            ],
          }),
        ],
        [CONTENT_WIDTH],
        { width: CONTENT_WIDTH, borders: tableBorder(BorderStyle.NONE) }
      )
    );

    nodes.push(...renderRepeatScalarGrid(field.itemFields, record));

    field.itemFields
      .filter((nestedField) => !isScalarField(nestedField))
      .forEach((nestedField) => {
        nodes.push(...renderField(nestedField, record[nestedField.key], 0, context));
      });
  });

  return nodes;
}

function renderRichField(field: RichFieldDef, value: unknown): Array<Paragraph | Table> {
  if (!Array.isArray(value)) {
    return [fieldParagraph("N/A", { color: C.gray, italic: true, size: 18 })];
  }

  return renderRichBlocks(value);
}

export function renderField(field: FieldDef, value: unknown, depth = 0, context: RenderContext = {}): Array<Paragraph | Table> {
  void depth;
  switch (field.type) {
    case "text":
    case "textarea":
    case "number":
    case "date":
    case "select":
    case "checkbox":
      return [renderScalarTable(field, value)];
    case "table":
      return renderTableField(field, value);
    case "rich":
      return renderRichField(field, value);
    case "repeat":
      return renderRepeatField(field, value, context);
  }

  throw new Error("DOCGEN_UNSUPPORTED_FIELD");
}
