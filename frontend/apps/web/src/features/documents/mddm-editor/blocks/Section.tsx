import { createReactBlockSpec } from "@blocknote/react";
import styles from "./Section.module.css";
import { SectionExternalHTML } from "../engine/external-html";
import { defaultLayoutTokens } from "../engine/layout-ir";

export const Section = createReactBlockSpec(
  {
    type: "section",
    propSchema: {
      title: { default: "" },
      color: { default: "#6b1f2a" },
      locked: { default: true },
      optional: { default: false },
      variant: { default: "bar" },
      __template_block_id: { default: "" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div
        className={styles.section}
        data-mddm-block="section"
        data-variant={props.block.props.variant || "bar"}
        data-locked={props.block.props.locked}
      >
        <div className={styles.sectionHeader}>
          <span className={styles.sectionTitle}>
            {props.block.props.title || "Section"}
          </span>
          {props.block.props.optional ? (
            <span className={styles.optionalBadge}>Opcional</span>
          ) : null}
        </div>
      </div>
    ),
    toExternalHTML: ({ block, editor }) => {
      const sectionIndex = (editor.document as any[])
        .filter((b: any) => b.type === "section")
        .findIndex((b: any) => b.id === block.id);
      const sectionNumber = sectionIndex >= 0 ? sectionIndex + 1 : undefined;

      return (
        <SectionExternalHTML
          title={(block.props as { title?: string }).title ?? ""}
          tokens={defaultLayoutTokens}
          sectionNumber={sectionNumber}
        />
      );
    },
  },
);
