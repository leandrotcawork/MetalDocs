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
import { mmToTwip, ptToHalfPt } from "../../helpers/units";
import { interpretSection } from "../../layout-interpreter";
import type { MDDMBlock } from "../../../adapter";

function hexToFill(hex: string): string {
  // docx.js accepts hex strings without the leading "#". Uppercase for OOXML
  // canonical form and deterministic golden-fixture diffs.
  return hex.replace(/^#/, "").toUpperCase();
}

function attachOptions<T, O extends object>(instance: T, options: O): T {
  (instance as unknown as { options: O }).options = options;
  return instance;
}

export function emitSection(block: MDDMBlock, tokens: LayoutTokens): Table[] {
  const vm = interpretSection(
    { props: block.props as Record<string, unknown> },
    tokens,
    { sectionIndex: 0 },
  );

  const fill = hexToFill(vm.headerBg);
  const fontSizePt = parseFloat(vm.headerFontSize); // e.g. "9pt" → 9
  const heightMm = parseFloat(vm.headerHeight);     // e.g. "10mm" → 10

  const textRunOptions = {
    text: vm.title,
    bold: vm.headerFontWeight === "bold",
    color: hexToFill(vm.headerColor),
    size: ptToHalfPt(isNaN(fontSizePt) ? 9 : fontSizePt),
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
      value: mmToTwip(isNaN(heightMm) ? 10 : heightMm),
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
