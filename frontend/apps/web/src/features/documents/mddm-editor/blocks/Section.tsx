import { createReactBlockSpec } from "@blocknote/react";

export const Section = createReactBlockSpec(
  {
    type: "section",
    propSchema: {
      title: { default: "" },
      color: { default: "#6b1f2a" },
      locked: { default: true },
      optional: { default: false },
      __template_block_id: { default: "" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div data-mddm-block="section">
        <strong>{props.block.props.title || "Section"}</strong>
        {props.block.props.optional ? <span>Opcional</span> : null}
      </div>
    ),
  },
);
