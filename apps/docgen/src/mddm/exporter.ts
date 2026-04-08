import { Document, HeadingLevel, Packer, Paragraph, TextRun } from "docx";
import type { InlineRun, MDDMBlock, MDDMEnvelope, MDDMExportRequest } from "./types.js";

function runToTextRun(run: InlineRun): TextRun {
  const marks = new Set((run.marks ?? []).map((mark) => mark.type));
  return new TextRun({
    text: run.text,
    bold: marks.has("bold"),
    italics: marks.has("italic"),
    underline: marks.has("underline") ? {} : undefined,
    strike: marks.has("strike"),
  });
}

function renderParagraph(block: MDDMBlock): Paragraph {
  const children = (block.children as InlineRun[] | undefined) ?? [];
  return new Paragraph({ children: children.map(runToTextRun) });
}

function renderHeading(block: MDDMBlock): Paragraph {
  const level = (block.props.level as number) ?? 2;
  const headingLevel = level === 1 ? HeadingLevel.HEADING_1 : level === 2 ? HeadingLevel.HEADING_2 : HeadingLevel.HEADING_3;
  const children = (block.children as InlineRun[] | undefined) ?? [];
  return new Paragraph({ heading: headingLevel, children: children.map(runToTextRun) });
}

function renderSection(block: MDDMBlock, path: number[]): Paragraph[] {
  const num = path.length === 0 ? 1 : path[path.length - 1] + 1;
  const title = (block.props.title as string) ?? "Section";
  const children: Paragraph[] = [new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun(title)] })];

  for (const child of (block.children as MDDMBlock[] | undefined) ?? []) {
    children.push(...renderBlock(child, [...path, num]));
  }

  return children;
}

function renderBlock(block: MDDMBlock, path: number[]): Paragraph[] {
  switch (block.type) {
    case "section":
      return renderSection(block, path);
    case "paragraph":
      return [renderParagraph(block)];
    case "heading":
      return [renderHeading(block)];
    default:
      return [new Paragraph({ children: [new TextRun(`[Unsupported block: ${block.type}]`)] })];
  }
}

function renderEnvelope(envelope: MDDMEnvelope): Paragraph[] {
  const children: Paragraph[] = [];
  for (const block of envelope.blocks) {
    children.push(...renderBlock(block, []));
  }
  return children;
}

export async function exportMDDMToDocx(req: MDDMExportRequest): Promise<Uint8Array> {
  const doc = new Document({
    sections: [
      {
        properties: {
          page: {
            margin: {
              top: 900,
              right: 900,
              bottom: 900,
              left: 900,
            },
          },
        },
        children: renderEnvelope(req.envelope),
      },
    ],
  });

  const buf = await Packer.toBuffer(doc);
  return new Uint8Array(buf);
}
