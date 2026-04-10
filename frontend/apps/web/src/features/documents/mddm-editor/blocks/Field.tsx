import { createReactBlockSpec } from "@blocknote/react";

export const Field = createReactBlockSpec(
  {
    type: "field",
    propSchema: {
      label: { default: "" },
      // Field currently renders BlockNote inline content only.
      valueMode: { default: "inline", values: ["inline"] as const },
      locked: { default: true },
      hint: { default: "" },
      __template_block_id: { default: "" },
    },
    content: "inline",
  },
  {
    render: (props) => (
      <div data-mddm-block="field">
        <strong>{props.block.props.label || "Field"}</strong>
        {props.block.props.hint ? <small>{props.block.props.hint}</small> : null}
        <div ref={props.contentRef} />
      </div>
    ),
  },
);
