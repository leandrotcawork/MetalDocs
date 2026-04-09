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
      <td data-mddm-block="dataTableCell">
        <div ref={props.contentRef} />
      </td>
    ),
  },
);
