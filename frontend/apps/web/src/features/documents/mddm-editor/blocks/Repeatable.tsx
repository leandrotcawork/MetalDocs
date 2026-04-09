import { createReactBlockSpec } from "@blocknote/react";

export const Repeatable = createReactBlockSpec(
  {
    type: "repeatable",
    propSchema: {
      label: { default: "" },
      itemPrefix: { default: "Item" },
      locked: { default: true },
      minItems: { default: 0 },
      maxItems: { default: 200 },
      __template_block_id: { default: undefined, type: "string" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div data-mddm-block="repeatable">
        <strong>{props.block.props.label || "Repeatable"}</strong>
      </div>
    ),
  },
);
