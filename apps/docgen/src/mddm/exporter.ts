import { BorderStyle, Document, ExternalHyperlink, HeadingLevel, Packer, Paragraph, ShadingType, Table, TextRun } from "docx";
import { renderDataTable } from "./render-data-table.js";
import { renderFieldGroup } from "./render-tables.js";
import type { InlineRun, MDDMBlock, MDDMEnvelope, MDDMExportRequest, MDDMTemplateTheme } from "./types.js";

function invalid(code: string): never {
  throw new Error(code);
}

type RenderedNode = Paragraph | Table;

type ExportTheme = Required<MDDMTemplateTheme>;

const DEFAULT_THEME: ExportTheme = {
  accent: "#6b1f2a",
  accentLight: "#f9f3f3",
  accentDark: "#3e1018",
  accentBorder: "#dfc8c8",
};

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

function normalizeHex(value: string): string {
  return value.replace(/^#/, "").toUpperCase();
}

function hexToDocx(value: string): string {
  return normalizeHex(value);
}

function resolveTheme(theme?: MDDMTemplateTheme): ExportTheme {
  return {
    accent: typeof theme?.accent === "string" ? theme.accent : DEFAULT_THEME.accent,
    accentLight: typeof theme?.accentLight === "string" ? theme.accentLight : DEFAULT_THEME.accentLight,
    accentDark: typeof theme?.accentDark === "string" ? theme.accentDark : DEFAULT_THEME.accentDark,
    accentBorder: typeof theme?.accentBorder === "string" ? theme.accentBorder : DEFAULT_THEME.accentBorder,
  };
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

function validateTemplateTheme(theme: unknown): asserts theme is MDDMTemplateTheme {
  if (!isObject(theme)) {
    invalid("DOCGEN_INVALID_REQUEST");
  }

  const keys: (keyof MDDMTemplateTheme)[] = ["accent", "accentLight", "accentDark", "accentBorder"];
  for (const key of keys) {
    const value = theme[key];
    if (value !== undefined && typeof value !== "string") {
      invalid("DOCGEN_INVALID_REQUEST");
    }
  }
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

  if (input.templateTheme !== undefined) {
    validateTemplateTheme(input.templateTheme);
  }

  const request: MDDMExportRequest = {
    envelope: input.envelope,
    metadata: {
      document_code: metadata.document_code,
      title: metadata.title,
      revision_label: metadata.revision_label,
      mode: metadata.mode,
    },
    templateTheme: input.templateTheme,
  };

  return request;
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

function inlineRunToDocx(run: InlineRun): TextRun | ExternalHyperlink {
  const textRun = runToTextRun(run);
  if (run.link?.href) {
    return new ExternalHyperlink({
      children: [textRun],
      link: run.link.href,
    });
  }
  return textRun;
}

function renderParagraph(block: MDDMBlock): Paragraph {
  const children = (block.children as InlineRun[] | undefined) ?? [];
  return new Paragraph({ children: children.map(inlineRunToDocx) });
}

function renderHeading(block: MDDMBlock): Paragraph {
  const level = (block.props.level as number) ?? 2;
  const headingLevel = level === 1 ? HeadingLevel.HEADING_1 : level === 2 ? HeadingLevel.HEADING_2 : HeadingLevel.HEADING_3;
  const children = (block.children as InlineRun[] | undefined) ?? [];
  return new Paragraph({ heading: headingLevel, children: children.map(inlineRunToDocx) });
}

function renderListItem(block: MDDMBlock, isNumbered: boolean): Paragraph {
  const children = (block.children as InlineRun[] | undefined) ?? [];
  const level = (block.props.level as number) ?? 0;

  return new Paragraph({
    children: children.map(inlineRunToDocx),
    bullet: isNumbered ? undefined : { level },
    numbering: isNumbered ? { reference: "default-numbering", level } : undefined,
  });
}

function renderCode(block: MDDMBlock): Paragraph {
  const children = Array.isArray(block.children) ? block.children : [];
  return new Paragraph({
    shading: { fill: "F4F4F4" },
    children: children.map((child) => new TextRun({ text: (child as InlineRun).text, font: "Courier New" })),
  });
}

function renderDivider(): Paragraph {
  return new Paragraph({
    border: {
      bottom: {
        color: "999999",
        space: 1,
        style: BorderStyle.SINGLE,
        size: 6,
      },
    },
    children: [],
  });
}

function renderRichBlock(block: MDDMBlock, theme: ExportTheme, sectionPath: number[]): RenderedNode[] {
  const label = (block.props.label as string) ?? "";
  const out: RenderedNode[] = [
    new Paragraph({ children: [new TextRun({ text: label, bold: true })] }),
  ];
  for (const child of (block.children as MDDMBlock[]) ?? []) {
    out.push(...renderBlock(child, theme, sectionPath));
  }
  return out;
}

function renderQuote(block: MDDMBlock): Paragraph[] {
  const paragraphs = (block.children as MDDMBlock[]) ?? [];
  return paragraphs.map((p) => {
    const runs = (p.children as InlineRun[] | undefined) ?? [];
    return new Paragraph({
      indent: { left: 720 },
      children: runs.map(inlineRunToDocx),
    });
  });
}

function renderImagePlaceholder(block: MDDMBlock): Paragraph {
  const alt = (block.props.alt as string) ?? "";
  const caption = (block.props.caption as string) ?? "";
  const text = caption ? `[Image: ${alt}] ${caption}` : `[Image: ${alt}]`;
  return new Paragraph({
    children: [new TextRun({ text, italics: true })],
  });
}

function renderRepeatable(block: MDDMBlock, theme: ExportTheme, sectionPath: number[]): Paragraph[] {
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
      out.push(...(renderBlock(b, theme, [...sectionPath, idx + 1]) as Paragraph[]));
    }
  });
  return out;
}

function renderSection(block: MDDMBlock, num: number, theme: ExportTheme, sectionPath: number[] = [num]): RenderedNode[] {
  const title = (block.props.title as string) ?? "";
  const children: RenderedNode[] = [
    new Paragraph({
      heading: HeadingLevel.HEADING_1,
      shading: {
        type: ShadingType.CLEAR,
        fill: hexToDocx(theme.accent),
      },
      border: {
        bottom: {
          color: hexToDocx(theme.accentBorder),
          space: 1,
          style: BorderStyle.SINGLE,
          size: 4,
        },
      },
      children: [
        new TextRun({
          text: `${num}. ${title}`,
          bold: true,
          color: hexToDocx(theme.accentLight),
        }),
      ],
    }),
  ];

  let sectionNumber = 1;
  for (const child of (block.children as MDDMBlock[] | undefined) ?? []) {
    if (child.type === "section") {
      children.push(...renderSection(child, sectionNumber, theme, [...sectionPath, sectionNumber]));
      sectionNumber++;
      continue;
    }

    children.push(...renderBlock(child, theme, sectionPath));
  }

  return children;
}

function renderBlock(block: MDDMBlock, theme: ExportTheme, sectionPath: number[] = []): RenderedNode[] {
  switch (block.type) {
    case "section":
      return renderSection(block, 1, theme, sectionPath.length > 0 ? [...sectionPath, 1] : [1]);
    case "fieldGroup":
      return [renderFieldGroup(block, theme)];
    case "dataTable":
      return [renderDataTable(block, theme)];
    case "paragraph":
      return [renderParagraph(block)];
    case "heading":
      return [renderHeading(block)];
    case "bulletListItem":
      return [renderListItem(block, false)];
    case "numberedListItem":
      return [renderListItem(block, true)];
    case "code":
      return [renderCode(block)];
    case "divider":
      return [renderDivider()];
    case "repeatable":
      return renderRepeatable(block, theme, sectionPath);
    case "repeatableItem":
      return [];
    case "richBlock":
      return renderRichBlock(block, theme, sectionPath);
    case "quote":
      return renderQuote(block);
    case "image":
      return [renderImagePlaceholder(block)];
    default:
      return [new Paragraph({ children: [new TextRun(`[Unsupported block: ${block.type}]`)] })];
  }
}

function renderEnvelope(envelope: MDDMEnvelope, theme: ExportTheme): RenderedNode[] {
  const children: RenderedNode[] = [];
  let sectionNumber = 1;
  for (const block of envelope.blocks) {
    if (block.type === "section") {
      children.push(...renderSection(block, sectionNumber, theme, [sectionNumber]));
      sectionNumber++;
      continue;
    }

    children.push(...renderBlock(block, theme, []));
  }
  return children;
}

export async function exportMDDMToDocx(req: MDDMExportRequest): Promise<Uint8Array> {
  const runtime = normalizeMDDMExportRequest(req);
  const theme = resolveTheme(runtime.templateTheme);
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
        children: renderEnvelope(runtime.envelope, theme),
      },
    ],
  });

  const buf = await Packer.toBuffer(doc);
  return new Uint8Array(buf);
}
