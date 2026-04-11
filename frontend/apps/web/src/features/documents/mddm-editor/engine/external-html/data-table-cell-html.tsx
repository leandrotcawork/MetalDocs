import type { ReactNode } from "react";
import type { LayoutTokens } from "../layout-ir";

export type DataTableCellExternalHTMLProps = {
  tokens: LayoutTokens;
  children?: ReactNode;
};

export function DataTableCellExternalHTML({ tokens, children }: DataTableCellExternalHTMLProps) {
  return (
    <td
      className="mddm-data-table-cell"
      data-mddm-block="dataTableCell"
      style={{
        padding: `${tokens.spacing.cellPaddingMm}mm`,
        border: `0.5pt solid ${tokens.theme.accentBorder}`,
        verticalAlign: "top",
      }}
    >
      {children}
    </td>
  );
}
