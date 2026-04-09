import { createReactBlockSpec } from "@blocknote/react";

export const DataTableRow = createReactBlockSpec(
  {
    type: "dataTableRow",
    propSchema: {},
    content: "none",
  },
  {
    render: () => <tr data-mddm-block="dataTableRow" />,
  },
);
