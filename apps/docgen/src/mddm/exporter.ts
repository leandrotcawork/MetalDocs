import { Document, HeadingLevel, Packer, Paragraph, Table, TextRun } from "docx";
import { renderFieldGroup } from "./render-tables.js";
import type { InlineRun, MDDMBlock, MDDMEnvelope, MDDMExportRequest } from "./types.js";

function invalid(code: string): never {
  throw new Error(code);
}

type RenderedNode = Paragraph | Table;

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isInlineRun(value: unknown): value is InlineRun {
  return isObject(value) && typeof value.text === "string";
}

function validateInlineRun(run: unknown): void {
  if (!isInlineRun(run)) {
    invalid("DOCGEN_INVALID_REQUEST");
  }

  if (run.marks !== undefined) {
    if (!Array.isArray(run.marks)) {
      invalid("DOCGEN_INVALID_REQUEST");
    }
    run.marks.forEach((mark) => {
      if (!isObject(mark) || typeof mark.type !== "string") {
        invalid("DOCGEN_INVALID_REQUEST");
      }
    });
  }

  if (run.link !== undefined) {
    if (!isObject(run.link) || typeof run.link.href !== "string") {
      invalid("DOCGEN_INVALID_REQUEST");
    }
    if (run.link.title !== undefined && typeof run.link.title !== "string") {
      invalid("DOCGEN_INVALID_REQUEST");
    }
  }

  if (run.document_ref !== undefined) {
    if (
      !isObject(run.document_ref) ||
      typeof run.document_ref.target_document_id !== "string"
    ) {
      invalid("DOCGEN_INVALID_REQUEST");
    }
    if (
      run.document_ref.target_revision_label !== undefined &&
      typeof run.document_ref.target_revision_label !== "string"
    ) {
      invalid("DOCGEN_INVALID_REQUEST");
    }
  }
}

function isBlockArray(value: unknown): value is MDDMBlock[] {
  return Array.isArray(value) && value.every((child) => isObject(child) && typeof child.type === "string" && isObject(child.props));
}

function isInlineRunArray(value: unknown): value is InlineRun[] {
  return Array.isArray(value) && value.every((child) => isInlineRun(child));
}

function validateMDDMBlockChildren(block: MDDMBlock): void {
  if (!Array.isArray(block.children)) {
    return;
  }

  if (block.type === "fieldGroup") {
    if (!isBlockArray(block.children)) {
      invalid("DOCGEN_INVALID_REQUEST");
    }
    block.children.forEach((child) => {
      if (child.type !== "field") {
        invalid("DOCGEN_INVALID_REQUEST");
      }
      validateMDDMBlock(child);
    });
    return;
  }

  if (block.type === "field") {
    const valueMode = block.props.valueMode === "multiParagraph" ? "multiParagraph" : "inline";
    if (valueMode === "inline") {
      if (!isInlineRunArray(block.children)) {
        invalid("DOCGEN_INVALID_REQUEST");
      }
      block.children.forEach((child) => validateInlineRun(child));
      return;
    }

    if (!isBlockArray(block.children)) {
      invalid("DOCGEN_INVALID_REQUEST");
    }
    block.children.forEach((child) => validateMDDMBlock(child));
    return;
  }

  if (block.type === "section") {
    if (!isBlockArray(block.children)) {
      invalid("DOCGEN_INVALID_REQUEST");
    }
    block.children.forEach((child) => validateMDDMBlock(child));
    return;
  }

  if (block.type === "paragraph" || block.type === "heading") {
    block.children.forEach((child) => validateInlineRun(child));
  }
}

function validateMDDMBlock(block: unknown): void {
  if (!isObject(block) || typeof block.type !== "string" || !isObject(block.props)) {
    invalid("DOCGEN_INVALID_REQUEST");
  }

  if (block.type === "fieldGroup") {
    const columns = (block.props as Record<string, unknown>).columns;
    if (columns !== undefined && columns !== 1 && columns !== 2) {
      invalid("DOCGEN_INVALID_REQUEST");
    }
  }

  if (block.children !== undefined) {
    if (!Array.isArray(block.children)) {
      invalid("DOCGEN_INVALID_REQUEST");
    }
  }

  validateMDDMBlockChildren(block as MDDMBlock);
}

function validateMDDMEnvelope(envelope: unknown): asserts envelope is MDDMEnvelope {
  if (
    !isObject(envelope) ||
    typeof envelope.mddm_version !== "number" ||
    !Array.isArray(envelope.blocks)
  ) {
    invalid("DOCGEN_INVALID_REQUEST");
  }

  envelope.blocks.forEach((block) => validateMDDMBlock(block));
}

function normalizeMDDMExportRequest(input: unknown): MDDMExportRequest {
  if (!isObject(input) || !isObject(input.metadata)) {
    invalid("DOCGEN_INVALID_REQUEST");
  }

  const metadata = input.metadata;
  if (
    typeof metadata.document_code !== "string" ||
    typeof metadata.title !== "string" ||
    typeof metadata.revision_label !== "string" ||
    (metadata.mode !== "production" && metadata.mode !== "debug")
  ) {
    invalid("DOCGEN_INVALID_REQUEST");
  }

  validateMDDMEnvelope(input.envelope);

  return {
    envelope: input.envelope,
    metadata: {
      document_code: metadata.document_code,
      title: metadata.title,
      revision_label: metadata.revision_label,
      mode: metadata.mode,
    },
  };
}

function runToTextRun(run: InlineRun): TextRun {
  const marks = new Set(
    (run.marks ?? [])
      .filter((mark): mark is { type: string } => isObject(mark) && typeof mark.type === "string")
      .map((mark) => mark.type),
  );
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

function renderRepeatable(block: MDDMBlock, sectionPath: number[]): Paragraph[] {
  const items = (block.children as MDDMBlock[]) ?? [];
  const sectionNum = sectionPath[sectionPath.length - 1] ?? 0;
  const out: Paragraph[] = [];
  items.forEach((item, idx) => {
    const num = `${sectionNum}.${idx + 1}`;
    const title = (item.props.title as string) ?? "";
    out.push(
      new Paragraph({
        heading: HeadingLevel.HEADING_2,
        children: [new TextRun({ text: `${num} ${title}`, bold: true })],
      }),
    );
    const body = (item.children as MDDMBlock[]) ?? [];
    for (const b of body) {
      out.push(...(renderBlock(b, [...sectionPath, idx + 1]) as Paragraph[]));
    }
  });
  return out;
}

function renderSection(block: MDDMBlock, num: number, sectionPath: number[] = [num]): RenderedNode[] {
  const title = (block.props.title as string) ?? "";
  const children: RenderedNode[] = [new Paragraph({ heading: HeadingLevel.HEADING_1, children: [new TextRun({ text: `${num}. ${title}`, bold: true })] })];

  let sectionNumber = 1;
  for (const child of (block.children as MDDMBlock[] | undefined) ?? []) {
    if (child.type === "section") {
      children.push(...renderSection(child, sectionNumber, [...sectionPath, sectionNumber]));
      sectionNumber++;
      continue;
    }

    children.push(...renderBlock(child, sectionPath));
  }

  return children;
}

function renderBlock(block: MDDMBlock, sectionPath: number[] = []): RenderedNode[] {
  switch (block.type) {
    case "section":
      return renderSection(block, 1, sectionPath.length > 0 ? [...sectionPath, 1] : [1]);
    case "fieldGroup":
      return [renderFieldGroup(block)];
    case "paragraph":
      return [renderParagraph(block)];
    case "heading":
      return [renderHeading(block)];
    case "repeatable":
      return renderRepeatable(block, sectionPath);
    case "repeatableItem":
      return [];
    default:
      return [new Paragraph({ children: [new TextRun(`[Unsupported block: ${block.type}]`)] })];
  }
}

function renderEnvelope(envelope: MDDMEnvelope): RenderedNode[] {
  const children: RenderedNode[] = [];
  let sectionNumber = 1;
  for (const block of envelope.blocks) {
    if (block.type === "section") {
      children.push(...renderSection(block, sectionNumber, [sectionNumber]));
      sectionNumber++;
      continue;
    }

    children.push(...renderBlock(block, []));
  }
  return children;
}

export async function exportMDDMToDocx(req: MDDMExportRequest): Promise<Uint8Array> {
  const runtime = normalizeMDDMExportRequest(req);
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
        children: renderEnvelope(runtime.envelope),
      },
    ],
  });

  const buf = await Packer.toBuffer(doc);
  return new Uint8Array(buf);
}
