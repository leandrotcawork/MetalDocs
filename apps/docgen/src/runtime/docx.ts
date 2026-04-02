import {
  AlignmentType,
  BorderStyle,
  ImportedXmlComponent,
  Paragraph,
  TableLayoutType,
  type IParagraphOptions,
  type IRunOptions,
  type ITableBordersOptions,
  type ITableCellBorders,
  type ITableCellOptions,
  ShadingType,
  Table,
  TableCell,
  TableRow,
  TextRun,
  VerticalAlign,
  WidthType,
} from "docx";

export const C = {
  purple: "4A3DB5",
  purpleLight: "EEEDFE",
  teal: "0F6E56",
  tealLight: "E1F5EE",
  amber: "BA7517",
  amberLight: "FAEEDA",
  coral: "993C1D",
  coralLight: "FAECE7",
  blue: "185FA5",
  blueLight: "E6F1FB",
  gray: "444441",
  grayLight: "F1EFE8",
  grayMid: "D3D1C7",
  white: "FFFFFF",
  border: "CCCCCC",
} as const;

export const PAGE_WIDTH = 12240;
export const PAGE_HEIGHT = 15840;
export const PAGE_MARGIN = 900;
export const CONTENT_WIDTH = 9360;
export const DEFAULT_FONT = "Arial";
export const DEFAULT_FONT_SIZE = 20;
export const HEADER_TITLE_WIDTH = 6000;
export const HEADER_META_WIDTH = CONTENT_WIDTH - HEADER_TITLE_WIDTH;
export const HEADER_META_COLUMN_WIDTH = HEADER_META_WIDTH / 2;
export const HEADER_ROW_1 = [HEADER_TITLE_WIDTH, HEADER_META_COLUMN_WIDTH, HEADER_META_COLUMN_WIDTH] as const;
export const HEADER_ROW_2 = [HEADER_TITLE_WIDTH, HEADER_META_WIDTH] as const;
export const LABEL_VALUE_ROW = [2200, 7160] as const;
export const REPEAT_ROW = [4680, 4680] as const;
export const PAIRED_SCALAR_WIDTHS = [2200, 2480, 2200, 2480] as const;

export const emptyBorder: ITableBordersOptions = { top: { style: BorderStyle.NONE }, bottom: { style: BorderStyle.NONE }, left: { style: BorderStyle.NONE }, right: { style: BorderStyle.NONE } };

export function cellBorder(style: (typeof BorderStyle)[keyof typeof BorderStyle], color?: string): ITableCellBorders {
  const border = { style, size: style === BorderStyle.NONE ? 0 : 4, color: color ? normalizeHex(color) : undefined };
  return {
    top: border,
    bottom: border,
    left: border,
    right: border,
    start: border,
    end: border,
  };
}

export function tableBorder(style: (typeof BorderStyle)[keyof typeof BorderStyle], color?: string): ITableBordersOptions {
  const border = { style, size: style === BorderStyle.NONE ? 0 : 4, color: color ? normalizeHex(color) : undefined };
  return {
    top: border,
    bottom: border,
    left: border,
    right: border,
  };
}

function normalizeHex(value: string): string {
  return value.replace(/^#/, "").toUpperCase();
}

// Maps each document section color to its pre-defined light variant from the design system.
// Falls back to mixWithWhite if the color is not in the palette.
const LIGHT_COLOR_MAP: Record<string, string> = {
  "4A3DB5": "EEEDFE", // purple  → purpleLight
  "0F6E56": "E1F5EE", // teal    → tealLight
  "BA7517": "FAEEDA", // amber   → amberLight
  "993C1D": "FAECE7", // coral   → coralLight
  "185FA5": "E6F1FB", // blue    → blueLight
  "444441": "F1EFE8", // gray    → grayLight
};

export function getLightColor(hexNoHash: string): string {
  const key = hexNoHash.replace(/^#/, "").toUpperCase();
  return LIGHT_COLOR_MAP[key] ?? mixWithWhite(key, 0.85);
}

export function mixWithWhite(color: string, whiteMix = 0.3): string {
  const hex = normalizeHex(color);
  const r = Number.parseInt(hex.slice(0, 2), 16);
  const g = Number.parseInt(hex.slice(2, 4), 16);
  const b = Number.parseInt(hex.slice(4, 6), 16);
  const mix = Math.max(0, Math.min(1, whiteMix));
  const blend = (channel: number) => Math.round(channel * (1 - mix) + 255 * mix);
  return [blend(r), blend(g), blend(b)].map((channel) => channel.toString(16).padStart(2, "0")).join("").toUpperCase();
}

export function run(text: string, options: Partial<IRunOptions> = {}): TextRun {
  return new TextRun({
    text,
    font: DEFAULT_FONT,
    size: DEFAULT_FONT_SIZE,
    ...options,
  });
}

export function paragraph(children: NonNullable<IParagraphOptions["children"]>, options: Omit<IParagraphOptions, "children"> = {}): Paragraph {
  return new Paragraph({
    ...options,
    children,
  });
}

type CellOptions = {
  width: number;
  columnSpan?: number;
  fill?: string;
  borders?: ITableCellBorders;
  alignment?: (typeof AlignmentType)[keyof typeof AlignmentType];
  verticalAlign?: "top" | "center" | "bottom";
  bold?: boolean;
  color?: string;
  size?: number;
  italic?: boolean;
  children: Array<Paragraph | Table>;
};

export function makeCell({
  width,
  columnSpan,
  fill,
  borders,
  alignment,
  verticalAlign = VerticalAlign.CENTER,
  bold,
  color,
  size,
  italic,
  children,
}: CellOptions): TableCell {
  return new TableCell({
    width: { size: width, type: WidthType.DXA },
    columnSpan,
    shading: {
      type: ShadingType.CLEAR,
      fill: fill ? normalizeHex(fill) : undefined,
    },
    borders: borders ?? cellBorder(BorderStyle.SINGLE, C.border),
    margins: {
      marginUnitType: WidthType.DXA,
      top: 80,
      bottom: 80,
      left: 140,
      right: 140,
    },
    verticalAlign,
    children: children.length
      ? children
      : [
          paragraph([
            run(" ", {
              bold,
              color,
              size,
              italics: italic,
            }),
          ], { alignment }),
        ],
  });
}

export function makeTable(rows: TableRow[], columnWidths: readonly number[], options: {
  borders?: ITableBordersOptions;
  width?: number;
} = {}): Table {
  return new Table({
    layout: TableLayoutType.FIXED,
    width: { size: options.width ?? CONTENT_WIDTH, type: WidthType.DXA },
    columnWidths,
    borders: options.borders ?? emptyBorder,
    rows,
  });
}

export function oneCellTable(cell: TableCell, width = CONTENT_WIDTH, columnWidths: readonly number[] = [CONTENT_WIDTH]): Table {
  return makeTable([new TableRow({ children: [cell] })], columnWidths, { width, borders: emptyBorder });
}

export function borderedTextCell(
  width: number,
  text: string,
  fill?: string,
  color?: string,
  size?: number,
  bold?: boolean,
  italic?: boolean,
  alignment?: (typeof AlignmentType)[keyof typeof AlignmentType]
): TableCell {
  return makeCell({
    width,
    fill,
    color,
    size,
    bold,
    italic,
    alignment,
    children: [paragraph([run(text, { bold, color, size, italics: italic })], { alignment })],
  });
}

export function fieldParagraph(text: string, options: {
  bold?: boolean;
  color?: string;
  size?: number;
  italic?: boolean;
  alignment?: (typeof AlignmentType)[keyof typeof AlignmentType];
} = {}): Paragraph {
  return paragraph([run(text, { bold: options.bold, color: options.color, size: options.size, italics: options.italic })], {
    alignment: options.alignment,
  });
}

export function makePageNumberField(): ImportedXmlComponent[] {
  return [
    ImportedXmlComponent.fromXmlString(`<w:r><w:fldChar w:fldCharType="begin"/></w:r>`),
    ImportedXmlComponent.fromXmlString(`<w:r><w:instrText xml:space="preserve"> PAGE </w:instrText></w:r>`),
    ImportedXmlComponent.fromXmlString(`<w:r><w:fldChar w:fldCharType="separate"/></w:r>`),
    ImportedXmlComponent.fromXmlString(`<w:r><w:t>1</w:t></w:r>`),
    ImportedXmlComponent.fromXmlString(`<w:r><w:fldChar w:fldCharType="end"/></w:r>`),
  ];
}

export function borderlessTopRule(): Paragraph {
  return paragraph([run(" ")], {
    border: {
      top: {
        style: BorderStyle.SINGLE,
        size: 4,
        color: C.grayMid,
      },
    },
  });
}
