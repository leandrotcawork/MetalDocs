import { createReactBlockSpec } from "@blocknote/react";

export const DataTableCell = createReactBlockSpec(
  {
    type: "dataTableCell",
    propSchema: {
      columnKey: { default: "" },
    },
    content: "inline",
  },
  {
    render: (props) => (
      <div data-mddm-block="dataTableCell">
        <span>{props.block.props.columnKey}</span>
        <div ref={props.contentRef} />
      </div>
    ),
  },
);

