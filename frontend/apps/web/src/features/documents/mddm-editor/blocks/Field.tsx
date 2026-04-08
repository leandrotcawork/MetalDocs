import { createReactBlockSpec } from "@blocknote/react";

export const Field = createReactBlockSpec(
  {
    type: "field",
    propSchema: {
      label: { default: "" },
      valueMode: { default: "inline", values: ["inline", "multiParagraph"] as const },
      locked: { default: true },
      __template_block_id: { default: undefined, type: "string" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div data-mddm-block="field">
        <strong>{props.block.props.label || "Field"}</strong>
      </div>
    ),
  },
);
