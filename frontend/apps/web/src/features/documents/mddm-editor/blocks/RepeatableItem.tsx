import { createReactBlockSpec } from "@blocknote/react";

export const RepeatableItem = createReactBlockSpec(
  {
    type: "repeatableItem",
    propSchema: {
      title: { default: "" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div data-mddm-block="repeatableItem">
        <strong>{props.block.props.title || "Item"}</strong>
      </div>
    ),
  },
);

