import { createReactBlockSpec } from "@blocknote/react";
import styles from "./RichBlock.module.css";
import { RichBlockExternalHTML } from "../engine/external-html";
import { getEditorTokens } from "../engine/editor-tokens";
import { interpretRichBlock } from "../engine/layout-interpreter/rich-block-interpreter";

export const RichBlock = createReactBlockSpec(
  {
    type: "richBlock",
    propSchema: {
      label: { default: "" },
      locked: { default: true },
      chrome: { default: "labeled" },
      __template_block_id: { default: "" },
      styleJson: { default: "{}" },
      capabilitiesJson: { default: "{}" },
    },
    content: "none",
  },
  {
    render: (props) => {
      const tokens = getEditorTokens(props.editor);
      const vm = interpretRichBlock(
        { props: props.block.props as Record<string, unknown> },
        tokens,
      );

      return (
        <div
          className={styles.richBlock}
          data-mddm-block="richBlock"
          data-chrome={vm.chrome}
          data-locked={vm.locked}
          style={{
            "--mddm-richblock-label-bg": vm.labelBackground,
            "--mddm-richblock-label-color": vm.labelColor,
            "--mddm-richblock-label-font-size": vm.labelFontSize,
            "--mddm-richblock-border-color": vm.borderColor,
          } as React.CSSProperties}
        >
          <div className={styles.richBlockHeader}>
            <strong>{vm.label || "Rich Block"}</strong>
          </div>
        </div>
      );
    },
    toExternalHTML: (props) => (
      <RichBlockExternalHTML
        tokens={getEditorTokens(props.editor)}
        label={props.block.props.label as string}
        chrome={props.block.props.chrome as string}
      />
    ),
  },
);
