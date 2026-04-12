import type { ReactNode } from "react";
import type { LayoutTokens } from "../layout-ir";

type RawCell = { type?: string; text?: string };
type RawRow = { cells?: RawCell[][] };
type RawTableContent = {
  type?: string;
  headerRows?: number;
  rows?: RawRow[];
};

export type DataTableExternalHTMLProps = {
  tokens: LayoutTokens;
  label?: string;
  tableContent?: unknown;
  children?: ReactNode;
};

export function DataTableExternalHTML({ tokens, label, tableContent }: DataTableExternalHTMLProps) {
  const content = tableContent as RawTableContent | undefined;
  const headerRows = content?.headerRows ?? 1;
  const rows = content?.rows ?? [];

  const headerStyle: React.CSSProperties = {
    padding: `${tokens.spacing.cellPaddingMm}mm ${tokens.spacing.cellPaddingMm * 1.5}mm`,
    background: tokens.theme.accentLight,
    color: tokens.theme.accentDark,
    fontWeight: 700,
    border: `0.5pt solid ${tokens.theme.accentBorder}`,
    textAlign: "left" as const,
    verticalAlign: "top" as const,
  };

  const cellStyle: React.CSSProperties = {
    padding: `${tokens.spacing.cellPaddingMm}mm ${tokens.spacing.cellPaddingMm * 1.5}mm`,
    border: `0.5pt solid ${tokens.theme.accentBorder}`,
    verticalAlign: "top" as const,
  };

  const tableHeaderBarStyle: React.CSSProperties = {
    padding: `${tokens.spacing.cellPaddingMm}mm ${tokens.spacing.cellPaddingMm * 1.5}mm`,
    background: tokens.theme.accentLight,
    fontWeight: 700,
    color: tokens.theme.accentDark,
    borderBottom: `0.5pt solid ${tokens.theme.accentBorder}`,
  };

  const getCellText = (cell: RawCell[]): string =>
    cell.map((run) => run.text ?? "").join("");

  return (
    <table
      data-mddm-block="dataTable"
      style={{
        width: "100%",
        borderCollapse: "collapse",
        border: `0.5pt solid ${tokens.theme.accentBorder}`,
        borderRadius: "4px",
        overflow: "hidden",
      }}
    >
      {label && (
        <thead>
          <tr>
            <th
              colSpan={rows[0]?.cells?.length ?? 1}
              style={tableHeaderBarStyle}
            >
              {label}
            </th>
          </tr>
        </thead>
      )}
      <tbody>
        {rows.map((row, rowIdx) => (
          <tr key={rowIdx}>
            {(row.cells ?? []).map((cell, cellIdx) => {
              const isHeader = rowIdx < headerRows;
              return isHeader ? (
                <th key={cellIdx} style={headerStyle}>
                  {getCellText(cell)}
                </th>
              ) : (
                <td key={cellIdx} style={cellStyle}>
                  {getCellText(cell)}
                </td>
              );
            })}
          </tr>
        ))}
      </tbody>
    </table>
  );
}
