import type { LayoutTokens } from "../layout-ir";

export type RepeatableItemExternalHTMLProps = {
  tokens: LayoutTokens;
  title?: string;
  sectionNumber?: number;
  itemNumber?: number;
};

export function RepeatableItemExternalHTML({ tokens, title, sectionNumber, itemNumber }: RepeatableItemExternalHTMLProps) {
  const n = itemNumber ?? 1;
  const prefix = sectionNumber && sectionNumber > 0 ? `${sectionNumber}.${n}` : `${n}.`;
  const displayTitle = title
    ? `${prefix} ${title}`
    : `Item ${prefix}`;

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
