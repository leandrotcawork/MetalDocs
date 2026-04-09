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
      <div
        data-mddm-block="dataTableCell"
        data-column-key={props.block.props.columnKey}
        role="cell"
      >
        <div ref={props.contentRef} />
      </div>
    ),
  },
);
