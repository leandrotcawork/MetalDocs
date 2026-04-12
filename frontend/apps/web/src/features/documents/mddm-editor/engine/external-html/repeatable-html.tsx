import type { LayoutTokens } from "../layout-ir";

export type RepeatableExternalHTMLProps = {
  tokens: LayoutTokens;
  label?: string;
};

export function RepeatableExternalHTML({ tokens, label }: RepeatableExternalHTMLProps) {
  return (
    <table
      data-mddm-block="repeatable"
      style={{
        width: "100%",
        borderCollapse: "collapse",
        border: `0.5pt solid ${tokens.theme.accentBorder}`,
        marginBottom: `${tokens.spacing.blockGapMm}mm`,
      }}
    >
      <thead>
        <tr>
          <th
            style={{
              padding: `${tokens.spacing.cellPaddingMm}mm`,
              background: tokens.theme.accentLight,
              color: tokens.theme.accentDark,
              fontWeight: 700,
              borderBottom: `0.5pt solid ${tokens.theme.accentBorder}`,
              textAlign: "left" as const,
            }}
          >
            {label ?? "Repeatable"}
          </th>
        </tr>
      </thead>
    </table>
  );
}
