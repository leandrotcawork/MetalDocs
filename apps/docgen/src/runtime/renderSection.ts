import { AlignmentType, BorderStyle, Paragraph, Table, TableRow } from "docx";
import { renderScalarValue } from "./blocks.js";
import { renderField } from "./renderField.js";
import {
  C,
  CONTENT_WIDTH,
  PAIRED_SCALAR_WIDTHS,
  cellBorder,
  fieldParagraph,
  makeCell,
  makeTable,
  tableBorder,
} from "./docx.js";
import type { FieldDef, SectionDef, SectionValues } from "./types.js";

// Short scalar field types that pair well 2-up in a single row.
const PAIR_TYPES = new Set(["text", "date", "number", "select", "checkbox"]);

function isPairable(field: FieldDef): boolean {
  return PAIR_TYPES.has(field.type);
}

function renderPairedRow(
  left: FieldDef,
  right: FieldDef,
  values: SectionValues
): Table {
  const [lw, vw] = [PAIRED_SCALAR_WIDTHS[0], PAIRED_SCALAR_WIDTHS[1]];
  return makeTable(
    [
      new TableRow({
        children: [
          makeCell({
            width: lw,
            fill: C.grayLight,
            children: [fieldParagraph(left.label, { bold: true, color: C.gray, size: 18 })],
          }),
          makeCell({
            width: vw,
            children: [fieldParagraph(renderScalarValue(values[left.key]), { italic: true, color: "2C2C2A", size: 18 })],
          }),
          makeCell({
            width: lw,
            fill: C.grayLight,
            children: [fieldParagraph(right.label, { bold: true, color: C.gray, size: 18 })],
          }),
          makeCell({
            width: vw,
            children: [fieldParagraph(renderScalarValue(values[right.key]), { italic: true, color: "2C2C2A", size: 18 })],
          }),
        ],
      }),
    ],
    PAIRED_SCALAR_WIDTHS,
    { width: CONTENT_WIDTH, borders: tableBorder(BorderStyle.NONE) }
  );
}

export function renderSection(section: SectionDef, values: SectionValues): Array<Paragraph | Table> {
  const headingColor = (section.color ?? C.gray).replace(/^#/, "").toUpperCase();
  const heading = makeTable(
    [
      new TableRow({
        children: [
          makeCell({
            width: CONTENT_WIDTH,
            fill: headingColor,
            borders: cellBorder(BorderStyle.NONE),
            alignment: AlignmentType.CENTER,
            children: [
              fieldParagraph(`${section.num} — ${section.title.toUpperCase()}`, {
                bold: true,
                color: C.white,
                size: 20,
                alignment: AlignmentType.CENTER,
              }),
            ],
          }),
        ],
      }),
    ],
    [CONTENT_WIDTH],
    { width: CONTENT_WIDTH, borders: tableBorder(BorderStyle.NONE) }
  );

  const nodes: Array<Paragraph | Table> = [heading];

  let i = 0;
  while (i < section.fields.length) {
    const field = section.fields[i];
    if (isPairable(field)) {
      const next = section.fields[i + 1];
      if (next && isPairable(next)) {
        nodes.push(renderPairedRow(field, next, values));
        i += 2;
        continue;
      }
    }
    nodes.push(...renderField(field, values[field.key], 0, { sectionColor: headingColor }));
    i++;
  }

  return nodes;
}
