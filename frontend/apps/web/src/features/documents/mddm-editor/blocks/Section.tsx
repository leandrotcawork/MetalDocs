import { createReactBlockSpec } from "@blocknote/react";
import styles from "./Section.module.css";
import { SectionExternalHTML } from "../engine/external-html";
import { getEditorTokens } from "../engine/editor-tokens";
import { interpretSection } from "../engine/layout-interpreter/section-interpreter";

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
      styleJson: { default: "{}" },
      capabilitiesJson: { default: "{}" },
    },
    content: "none",
  },
  {
    render: (props) => {
      const tokens = getEditorTokens(props.editor);
      const sectionIndex = (props.editor.document as any[])
        .filter((b: any) => b.type === "section")
        .findIndex((b: any) => b.id === props.block.id);
      const vm = interpretSection(
        { props: props.block.props as Record<string, unknown> },
        tokens,
        { sectionIndex: sectionIndex >= 0 ? sectionIndex : 0 },
      );

      return (
        <div
          className={styles.section}
          data-mddm-block="section"
          data-variant={props.block.props.variant || "bar"}
          data-locked={vm.locked}
          style={{
            height: vm.headerHeight,
            background: vm.headerBg,
            color: vm.headerColor,
            fontSize: vm.headerFontSize,
            fontWeight: vm.headerFontWeight,
          }}
        >
          <div className={styles.sectionHeader}>
            <span className={styles.sectionTitle}>{vm.title}</span>
            {vm.optional ? (
              <span className={styles.optionalBadge}>Opcional</span>
            ) : null}
          </div>
        </div>
      );
    },
    toExternalHTML: ({ block, editor }) => {
      const tokens = getEditorTokens(editor);
      const sectionIndex = (editor.document as any[])
        .filter((b: any) => b.type === "section")
        .findIndex((b: any) => b.id === block.id);
      const vm = interpretSection(
        { props: block.props as Record<string, unknown> },
        tokens,
        { sectionIndex: sectionIndex >= 0 ? sectionIndex : 0 },
      );

      return (
        <SectionExternalHTML
          title={vm.title}
          tokens={tokens}
          sectionNumber={parseInt(vm.number, 10)}
        />
      );
    },
  },
);
