import {
  Table,
  TableRow,
  TableCell,
  Paragraph,
  TextRun,
  WidthType,
  BorderStyle,
  HeightRule,
} from "docx";
import type { LayoutTokens } from "../../layout-ir";
import { defaultComponentRules } from "../../layout-ir";
import { mmToTwip, ptToHalfPt } from "../../helpers/units";
import type { MDDMBlock } from "../../../adapter";
import { mddmTextRunsToDocxRuns } from "../inline-content";
import { extractTextRuns } from "./paragraph";

function hexToFill(hex: string): string {
  return hex.replace(/^#/, "").toUpperCase();
}

function attachOptions<T, O extends object>(instance: T, options: O): T {
  (instance as unknown as { options: O }).options = options;
  return instance;
}

export function emitField(block: MDDMBlock, tokens: LayoutTokens): Table[] {
  const rule = defaultComponentRules.field;
  const label = (block.props as { label?: string }).label ?? "";
  const labelFill = hexToFill(tokens.theme.accentLight);
  const borderColor = hexToFill(tokens.theme.accentBorder);

  const borders = {
    top: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    left: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    right: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
  } as const;

  const labelRunOptions = {
    text: label,
    size: ptToHalfPt(rule.labelFontSizePt),
    font: tokens.typography.exportFont,
  } as const;
  const labelRun = attachOptions(new TextRun(labelRunOptions), labelRunOptions);

  const labelParagraphOptions = { children: [labelRun] } as const;
  const labelParagraph = attachOptions(new Paragraph(labelParagraphOptions), labelParagraphOptions);

  const labelCellOptions = {
    width: {
      size: rule.labelWidthPercent,
      type: WidthType.PERCENTAGE,
    },
    shading: { fill: labelFill, type: "clear" as const, color: "auto" },
    borders,
    children: [labelParagraph],
  };
  const labelCell = attachOptions(new TableCell(labelCellOptions), labelCellOptions);

  const valueRuns = mddmTextRunsToDocxRuns(extractTextRuns(block), tokens);
  const valueParagraphOptions = {
    children: valueRuns.length > 0 ? valueRuns : [new TextRun({ text: "" })],
  };
  const valueParagraph = attachOptions(new Paragraph(valueParagraphOptions), valueParagraphOptions);

  const valueCellOptions = {
    width: {
      size: rule.valueWidthPercent,
      type: WidthType.PERCENTAGE,
    },
    borders,
    children: [valueParagraph],
  };
  const valueCell = attachOptions(new TableCell(valueCellOptions), valueCellOptions);

  const rowOptions = {
    height: {
      value: mmToTwip(rule.minHeightMm),
      rule: HeightRule.AT_LEAST,
    },
    children: [labelCell, valueCell],
  } as const;
  const row = attachOptions(new TableRow(rowOptions), rowOptions);

  const tableOptions = {
    width: { size: 100, type: WidthType.PERCENTAGE },
    rows: [row],
  } as const;
  const table = attachOptions(new Table(tableOptions), tableOptions);

  return [table];
}
