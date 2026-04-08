import { AlignmentType, Document, HeadingLevel, Packer, Paragraph, TextRun, UnderlineType } from "docx";
import type { ParagraphChild } from "docx";
import type { InlineRun, MDDMBlock, MDDMEnvelope, MDDMExportRequest } from "./types.js";

const CONTENT_TYPE = "application/vnd.openxmlformats-officedocument.wordprocessingml.document";

type RenderedNode = Paragraph;
type HeadingLevelValue = (typeof HeadingLevel)[keyof typeof HeadingLevel];

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isInlineRunArray(children: unknown): children is InlineRun[] {
  return Array.isArray(children) && children.every((child) => isObject(child) && typeof child.text === "string");
}

function isBlockArray(children: unknown): children is MDDMBlock[] {
  return Array.isArray(children) && children.every((child) => isObject(child) && typeof child.type === "string" && typeof child.id === "string");
}

function asString(value: unknown): string {
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return "";
}

function markSet(run: InlineRun): Set<string> {
  return new Set((run.marks ?? []).map((mark) => mark.type));
}

export function runToTextRun(run: InlineRun): TextRun {
  const marks = markSet(run);
  const styledRun = new TextRun({
    text: run.text,
    bold: marks.has("bold"),
    italics: marks.has("italic"),
    underline: marks.has("underline") ? { type: UnderlineType.SINGLE } : undefined,
    strike: marks.has("strike"),
    color: run.link || run.document_ref ? "0563C1" : undefined,
    style: run.link || run.document_ref ? "Hyperlink" : undefined,
  });

  return styledRun;
}

function renderInlineChildren(children: InlineRun[]): ParagraphChild[] {
  return children.map((run) => runToTextRun(run));
}

export function renderParagraph(block: MDDMBlock): RenderedNode[] {
  const inlineChildren = isInlineRunArray(block.children) ? renderInlineChildren(block.children) : [];
  const text = asString(block.props.text) || asString(block.props.content) || asString(block.props.body);

  if (inlineChildren.length > 0) {
    return [new Paragraph({ children: inlineChildren })];
  }

  return [new Paragraph({ children: [new TextRun(text)] })];
}

function headingLevelFromValue(value: unknown, fallback: HeadingLevelValue): HeadingLevelValue {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return fallback;
  }
  const level = Math.trunc(value);
  switch (level) {
    case 1:
      return HeadingLevel.HEADING_1;
    case 2:
      return HeadingLevel.HEADING_2;
    case 3:
      return HeadingLevel.HEADING_3;
    case 4:
      return HeadingLevel.HEADING_4;
    case 5:
      return HeadingLevel.HEADING_5;
    default:
      return HeadingLevel.HEADING_6;
  }
}

export function renderHeading(block: MDDMBlock, sectionPath: number[]): RenderedNode[] {
  const inlineChildren = isInlineRunArray(block.children) ? renderInlineChildren(block.children) : [];
  const title = inlineChildren.length > 0 ? inlineChildren : [new TextRun(asString(block.props.text) || asString(block.props.title) || "Heading")];
  const prefix = sectionPath.length > 0 ? `${sectionPath.join(".")} ` : "";
  const level = headingLevelFromValue(block.props.level, HeadingLevel.HEADING_1);

  return [
    new Paragraph({
      heading: level,
      children: prefix ? [new TextRun(prefix), ...title] : title,
    }),
  ];
}

export function renderSection(block: MDDMBlock, sectionPath: number[]): RenderedNode[] {
  const title = asString(block.props.title) || asString(block.props.label) || "Section";
  const inlineChildren = isInlineRunArray(block.children) ? renderInlineChildren(block.children) : [];
  const headingChildren = inlineChildren.length > 0 ? inlineChildren : [new TextRun(title)];

  const nodes: RenderedNode[] = [
    new Paragraph({
      heading: HeadingLevel.HEADING_1,
      children: sectionPath.length > 0 ? [new TextRun(`${sectionPath.join(".")} `), ...headingChildren] : headingChildren,
    }),
  ];

  if (isBlockArray(block.children)) {
    block.children.forEach((child, index) => {
      nodes.push(...renderBlock(child, [...sectionPath, index + 1]));
    });
    return nodes;
  }

  if (inlineChildren.length > 0) {
    nodes.push(new Paragraph({ children: inlineChildren }));
  }

  return nodes;
}

export function renderBlock(block: MDDMBlock, sectionPath: number[]): RenderedNode[] {
  switch (block.type) {
    case "section":
      return renderSection(block, sectionPath);
    case "paragraph":
      return renderParagraph(block);
    case "heading":
      return renderHeading(block, sectionPath);
    default:
      return [new Paragraph({ children: [new TextRun(`[Unsupported block: ${block.type}]`)] })];
  }
}

export function renderEnvelope(envelope: MDDMEnvelope): RenderedNode[] {
  return envelope.blocks.flatMap((block, index) => renderBlock(block, [index + 1]));
}

export async function exportMDDMToDocx(req: MDDMExportRequest): Promise<Uint8Array> {
  const sections = renderEnvelope(req.envelope);

  const doc = new Document({
    sections: [
      {
        children: [
          new Paragraph({
            alignment: AlignmentType.CENTER,
            children: [
              new TextRun({
                text: req.metadata.title,
                bold: true,
                size: 28,
              }),
            ],
          }),
          new Paragraph({
            alignment: AlignmentType.CENTER,
            children: [
              new TextRun({
                text: `${req.metadata.document_code} | ${req.metadata.revision_label} | ${req.metadata.mode}`,
                size: 18,
              }),
            ],
          }),
          ...sections,
        ],
      },
    ],
  });

  return Packer.toBuffer(doc);
}
