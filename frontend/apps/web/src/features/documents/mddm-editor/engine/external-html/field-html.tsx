import type { ReactNode } from "react";
import type { LayoutTokens } from "../layout-ir";
import { defaultComponentRules } from "../layout-ir";

export type FieldExternalHTMLProps = {
  label: string;
  tokens: LayoutTokens;
  children?: ReactNode;
};

export function FieldExternalHTML({ label, tokens, children }: FieldExternalHTMLProps) {
  const rule = defaultComponentRules.field;
  const borderStyle = `${rule.borderWidthPt}pt solid ${tokens.theme.accentBorder}`;

  return (
    <table
      className="mddm-field"
      data-mddm-block="field"
      style={{
        width: "100%",
        borderCollapse: "collapse",
        tableLayout: "fixed",
      }}
    >
      <tbody>
        <tr>
          <td
            style={{
              width: `${rule.labelWidthPercent}%`,
              background: tokens.theme.accentLight,
              fontSize: `${rule.labelFontSizePt}pt`,
              padding: `${tokens.spacing.cellPaddingMm}mm`,
              border: borderStyle,
              verticalAlign: "top",
              minHeight: `${rule.minHeightMm}mm`,
            }}
          >
            {label}
          </td>
          <td
            style={{
              width: `${rule.valueWidthPercent}%`,
              padding: `${tokens.spacing.cellPaddingMm}mm`,
              border: borderStyle,
              verticalAlign: "top",
              minHeight: `${rule.minHeightMm}mm`,
            }}
          >
            {children}
          </td>
        </tr>
      </tbody>
    </table>
  );
}
