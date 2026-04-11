import type { LayoutTokens } from "../layout-ir";
import { defaultComponentRules } from "../layout-ir";

export type SectionExternalHTMLProps = {
  title: string;
  tokens: LayoutTokens;
};

export function SectionExternalHTML({ title, tokens }: SectionExternalHTMLProps) {
  const rule = defaultComponentRules.section;

  return (
    <table
      className="mddm-section-header"
      data-mddm-block="section"
      style={{
        width: "100%",
        borderCollapse: "collapse",
        margin: `${tokens.spacing.blockGapMm}mm 0`,
      }}
    >
      <tbody>
        <tr>
          <td
            style={{
              background: tokens.theme.accent,
              height: `${rule.headerHeightMm}mm`,
              color: rule.headerFontColor,
              fontWeight: rule.headerFontWeight,
              fontSize: `${rule.headerFontSizePt}pt`,
              padding: "0 4mm",
              verticalAlign: "middle",
            }}
          >
            {title}
          </td>
        </tr>
      </tbody>
    </table>
  );
}
