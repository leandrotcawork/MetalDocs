import {
  Table,
  TableRow,
  TableCell,
  Paragraph,
  TextRun,
  WidthType,
  HeightRule,
  VerticalAlign,
} from "docx";
import type { LayoutTokens } from "../../layout-ir";
import { defaultComponentRules } from "../../layout-ir";
import { mmToTwip, ptToHalfPt } from "../../helpers/units";
import type { MDDMBlock } from "../../../adapter";

function hexToFill(hex: string): string {
  // docx.js accepts hex strings without the leading "#". We keep the original
  // casing so downstream tests and styling stay byte-identical to the token.
  return hex.replace(/^#/, "");
}

function attachOptions<T, O extends object>(instance: T, options: O): T {
  (instance as unknown as { options: O }).options = options;
  return instance;
}

export function emitSection(block: MDDMBlock, tokens: LayoutTokens): Table[] {
  const rule = defaultComponentRules.section;
  const title = (block.props as { title?: string }).title ?? "";
  const fill = hexToFill(tokens.theme.accent);

  const textRunOptions = {
    text: title,
    bold: rule.headerFontWeight === "bold",
    color: rule.headerFontColor.replace(/^#/, ""),
    size: ptToHalfPt(rule.headerFontSizePt),
    font: tokens.typography.exportFont,
  } as const;
  const titleRun = attachOptions(new TextRun(textRunOptions), textRunOptions);

  const paragraphOptions = { children: [titleRun] } as const;
  const titleParagraph = attachOptions(new Paragraph(paragraphOptions), paragraphOptions);

  const cellOptions = {
    shading: { fill, type: "clear" as const, color: "auto" },
    verticalAlign: VerticalAlign.CENTER,
    children: [titleParagraph],
  };
  const cell = attachOptions(new TableCell(cellOptions), cellOptions);

  const rowOptions = {
    height: {
      value: mmToTwip(rule.headerHeightMm),
      rule: HeightRule.EXACT,
    },
    children: [cell],
  } as const;
  const row = attachOptions(new TableRow(rowOptions), rowOptions);

  const tableOptions = {
    width: { size: 100, type: WidthType.PERCENTAGE },
    rows: [row],
  } as const;
  const table = attachOptions(new Table(tableOptions), tableOptions);

  return [table];
}
