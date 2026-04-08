import { Document, HeadingLevel, Packer, Paragraph, TextRun, UnderlineType } from "docx";
import type { ParagraphChild } from "docx";
import type { InlineRun, MDDMBlock, MDDMEnvelope, MDDMExportRequest } from "./types.js";

type RenderedNode = Paragraph;

function markSet(run: InlineRun): Set<string> {
  return new Set((run.marks ?? []).map((mark) => mark.type));
}

export function runToTextRun(run: InlineRun): TextRun {
  const marks = markSet(run);
  return new TextRun({
    text: run.text,
    bold: marks.has("bold"),
    italics: marks.has("italic"),
    underline: marks.has("underline") ? { type: UnderlineType.SINGLE } : undefined,
    strike: marks.has("strike"),
  });
}

function renderInlineChildren(children: InlineRun[]): ParagraphChild[] {
  return children.map((run) => runToTextRun(run));
}

export function renderParagraph(block: MDDMBlock): RenderedNode {
  const runs = (block.children as InlineRun[] | undefined) ?? [];
  return new Paragraph({ children: renderInlineChildren(runs) });
}

export function renderHeading(block: MDDMBlock): RenderedNode {
  const level = (block.props.level as number) ?? 2;
  const headingLevel =
    level === 1 ? HeadingLevel.HEADING_1 : level === 2 ? HeadingLevel.HEADING_2 : HeadingLevel.HEADING_3;

  return new Paragraph({
    heading: headingLevel,
    children: renderInlineChildren((block.children as InlineRun[] | undefined) ?? []),
  });
}

export function renderSection(block: MDDMBlock, _sectionPath: number[]): RenderedNode[] {
  const heading = new Paragraph({
    heading: HeadingLevel.HEADING_1,
    children: [new TextRun((block.props.title as string) ?? "Section")],
  });

  const nodes: RenderedNode[] = [heading];

  for (const child of (block.children as MDDMBlock[] | undefined) ?? []) {
    nodes.push(...renderBlock(child, []));
  }

  return nodes;
}

export function renderBlock(block: MDDMBlock, sectionPath: number[]): RenderedNode[] {
  switch (block.type) {
    case "section":
      return renderSection(block, sectionPath);
    case "paragraph":
      return [renderParagraph(block)];
    case "heading":
      return [renderHeading(block)];
    default:
      return [new Paragraph({ children: [new TextRun(`[Unsupported block: ${block.type}]`)] })];
  }
}

export function renderEnvelope(envelope: MDDMEnvelope): RenderedNode[] {
  const out: RenderedNode[] = [];
  for (const block of envelope.blocks) out.push(...renderBlock(block, []));
  return out;
}

export async function exportMDDMToDocx(req: MDDMExportRequest): Promise<Uint8Array> {
  const sections = renderEnvelope(req.envelope);

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
        children: [
          ...sections,
        ],
      },
    ],
  });

  return Packer.toBuffer(doc);
}
