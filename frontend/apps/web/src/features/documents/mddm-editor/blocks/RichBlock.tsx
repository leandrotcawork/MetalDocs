import { createReactBlockSpec } from "@blocknote/react";

export const RichBlock = createReactBlockSpec(
  {
    type: "richBlock",
    propSchema: {
      label: { default: "" },
      locked: { default: true },
      __template_block_id: { default: undefined, type: "string" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div data-mddm-block="richBlock">
        <strong>{props.block.props.label || "Rich Block"}</strong>
      </div>
    ),
  },
);
