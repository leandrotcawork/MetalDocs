import { createReactBlockSpec } from "@blocknote/react";

export const FieldGroup = createReactBlockSpec(
  {
    type: "fieldGroup",
    propSchema: {
      columns: { default: 1, values: [1, 2] as const },
      locked: { default: true },
      __template_block_id: { default: "" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div data-mddm-block="fieldGroup">
        <strong>Field Group</strong> ({props.block.props.columns} coluna(s))
      </div>
    ),
  },
);
