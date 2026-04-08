import { createReactBlockSpec } from "@blocknote/react";

function getColumnCount(columnsJson: string): number {
  try {
    const parsed = JSON.parse(columnsJson);
    return Array.isArray(parsed) ? parsed.length : 0;
  } catch {
    return 0;
  }
}

export const DataTable = createReactBlockSpec(
  {
    type: "dataTable",
    propSchema: {
      label: { default: "" },
      columnsJson: { default: "[]" },
      locked: { default: true },
      minRows: { default: 0 },
      maxRows: { default: 500 },
      __template_block_id: { default: undefined, type: "string" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div data-mddm-block="dataTable">
        <strong>{props.block.props.label || "Data Table"}</strong> (
        {getColumnCount(props.block.props.columnsJson)} colunas)
      </div>
    ),
  },
);
