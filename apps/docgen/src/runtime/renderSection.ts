import { AlignmentType, BorderStyle, Paragraph, Table, TableRow } from "docx";
import { renderField } from "./renderField.js";
import {
  C,
  CONTENT_WIDTH,
  cellBorder,
  fieldParagraph,
  makeCell,
  makeTable,
  tableBorder,
} from "./docx.js";
import type { SectionDef, SectionValues } from "./types.js";

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

  return [
    heading,
    ...section.fields.flatMap((field) => renderField(field, values[field.key], 0, { sectionColor: headingColor })),
  ];
}
