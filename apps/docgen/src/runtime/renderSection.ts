import { Paragraph, Table, TextRun } from "docx";
import { renderField } from "./renderField.js";
import type { SectionDef, SectionValues } from "./types.js";

export function renderSection(section: SectionDef, values: SectionValues): Array<Paragraph | Table> {
  const title = `${section.num}. ${section.title}`;
  const headingColor = section.color?.replace(/^#/, "");

  return [
    new Paragraph({
      children: [new TextRun({ text: title, bold: true, color: headingColor })],
      spacing: { before: 240, after: 80 },
    }),
    ...section.fields.flatMap((field) => renderField(field, values[field.key])),
  ];
}
