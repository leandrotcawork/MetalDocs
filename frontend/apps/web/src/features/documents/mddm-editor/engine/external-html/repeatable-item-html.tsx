import type { LayoutTokens } from "../layout-ir";

export type RepeatableItemExternalHTMLProps = {
  tokens: LayoutTokens;
  title?: string;
  itemNumber?: number;
};

export function RepeatableItemExternalHTML({ tokens, title, itemNumber }: RepeatableItemExternalHTMLProps) {
  const displayTitle = title && itemNumber
    ? `${itemNumber}. ${title}`
    : title ?? `Item ${itemNumber ?? 1}`;

  return (
    <table
      data-mddm-block="repeatableItem"
      style={{
        width: "100%",
        borderCollapse: "collapse",
        borderLeft: `3pt solid ${tokens.theme.accent}`,
        marginBottom: `${tokens.spacing.fieldGapMm}mm`,
      }}
    >
      <thead>
        <tr>
          <th
            style={{
              padding: `${tokens.spacing.cellPaddingMm}mm ${tokens.spacing.cellPaddingMm * 1.5}mm`,
              background: tokens.theme.accentLight,
              color: tokens.theme.accentDark,
              fontWeight: 700,
              textAlign: "left" as const,
            }}
          >
            {displayTitle}
          </th>
        </tr>
      </thead>
    </table>
  );
}
