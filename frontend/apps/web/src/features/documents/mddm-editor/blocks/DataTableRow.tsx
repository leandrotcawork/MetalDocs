import { createReactBlockSpec } from "@blocknote/react";

export const DataTableRow = createReactBlockSpec(
  {
    type: "dataTableRow",
    propSchema: {},
    content: "none",
  },
  {
    render: () => <div data-mddm-block="dataTableRow">Row</div>,
  },
);

