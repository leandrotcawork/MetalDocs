import { Document, Packer, Paragraph, TextRun } from "docx";

export async function generateDocx(_: unknown): Promise<Uint8Array> {
  const doc = new Document({
    sections: [
      {
        children: [
          new Paragraph({
            children: [new TextRun({ text: "MetalDocs Docgen Harness", bold: true })],
          }),
          new Paragraph("Document generated for harness validation."),
        ],
      },
    ],
  });

  return Packer.toBuffer(doc);
}
