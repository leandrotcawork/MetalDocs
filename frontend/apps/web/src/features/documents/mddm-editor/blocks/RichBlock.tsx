import { createReactBlockSpec } from "@blocknote/react";
import styles from "./RichBlock.module.css";
import { RichBlockExternalHTML } from "../engine/external-html";
import { getEditorTokens } from "../engine/editor-tokens";

export const RichBlock = createReactBlockSpec(
  {
    type: "richBlock",
    propSchema: {
      label: { default: "" },
      locked: { default: true },
      chrome: { default: "labeled" },
      __template_block_id: { default: "" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div
        className={styles.richBlock}
        data-mddm-block="richBlock"
        data-chrome={props.block.props.chrome || "labeled"}
      >
        <div className={styles.richBlockHeader}>
          <strong>{props.block.props.label || "Rich Block"}</strong>
        </div>
      </div>
    ),
    toExternalHTML: (props) => (
      <RichBlockExternalHTML
        tokens={getEditorTokens(props.editor)}
        label={props.block.props.label as string}
        chrome={props.block.props.chrome as string}
      />
    ),
  },
);
