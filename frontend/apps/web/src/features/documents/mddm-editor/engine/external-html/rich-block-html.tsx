import type { LayoutTokens } from "../layout-ir";

export type RichBlockExternalHTMLProps = {
  tokens: LayoutTokens;
  label?: string;
  chrome?: string;
};

export function RichBlockExternalHTML({ tokens, label, chrome }: RichBlockExternalHTMLProps) {
  const hasLabel = chrome === "labeled" && label;
  return (
    <table
      data-mddm-block="richBlock"
      style={{
        width: "100%",
        borderCollapse: "collapse",
        border: `0.5pt dashed ${tokens.theme.accentBorder}`,
        marginBottom: `${tokens.spacing.blockGapMm}mm`,
      }}
    >
      {hasLabel && (
        <thead>
          <tr>
            <th
              style={{
                padding: `${tokens.spacing.cellPaddingMm}mm`,
                color: tokens.theme.accentDark,
                background: "rgba(255,255,255,0.76)",
                borderBottom: `0.5pt solid ${tokens.theme.accentBorder}`,
                fontWeight: 700,
                textAlign: "left" as const,
              }}
            >
              {label}
            </th>
          </tr>
        </thead>
      )}
    </table>
  );
}
