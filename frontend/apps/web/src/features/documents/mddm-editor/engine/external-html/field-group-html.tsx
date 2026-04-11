import type { ReactNode } from "react";
import type { LayoutTokens } from "../layout-ir";

export type FieldGroupExternalHTMLProps = {
  columns: 1 | 2;
  tokens: LayoutTokens;
  children?: ReactNode;
};

export function FieldGroupExternalHTML({ columns, tokens, children }: FieldGroupExternalHTMLProps) {
  return (
    <table
      className="mddm-field-group"
      data-mddm-block="fieldGroup"
      data-columns={String(columns)}
      style={{
        width: "100%",
        borderCollapse: "collapse",
        margin: `${tokens.spacing.blockGapMm}mm 0`,
      }}
    >
      <tbody>
        <tr>
          <td style={{ padding: 0, verticalAlign: "top" }}>{children}</td>
        </tr>
      </tbody>
    </table>
  );
}
