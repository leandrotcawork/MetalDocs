import type { SchemaField } from "../contentSchemaTypes";

type TableFieldProps = {
  label: string;
  required?: boolean;
  rows: Record<string, unknown>[];
  columns: SchemaField[];
  onChange: (next: Record<string, unknown>[]) => void;
};

export function TableField({ label, required, rows, columns, onChange }: TableFieldProps) {
  return (
    <div className="content-builder-field">
      <span>
        {label}
        {required && <em className="content-builder-required">*</em>}
      </span>
      <div className="content-builder-table">
        <div className="content-builder-table-head">
          {columns.map((column) => (
            <span key={column.key}>{column.label ?? column.key}</span>
          ))}
          <span />
        </div>
        {rows.map((row, rowIndex) => (
          <div key={`${label}-${rowIndex}`} className="content-builder-table-row">
            {columns.map((column) => {
              const columnType = column.type ?? "text";
              const cellValue = (row?.[column.key] as string | number | undefined) ?? "";
              const handleCellChange = (nextValue: string | number) => {
                const nextRows = [...rows];
                const nextRow = { ...(rows[rowIndex] ?? {}), [column.key]: nextValue };
                nextRows[rowIndex] = nextRow;
                onChange(nextRows);
              };
              if (columnType === "select") {
                return (
                  <select
                    key={`${label}-${rowIndex}-${column.key}`}
                    value={String(cellValue ?? "")}
                    onChange={(event) => handleCellChange(event.target.value)}
                  >
                    <option value="">Selecione</option>
                    {(column.options ?? []).map((option) => (
                      <option key={option} value={option}>{option}</option>
                    ))}
                  </select>
                );
              }
              if (columnType === "number") {
                return (
                  <input
                    key={`${label}-${rowIndex}-${column.key}`}
                    type="number"
                    value={cellValue === "" ? "" : Number(cellValue)}
                    onChange={(event) =>
                      handleCellChange(event.target.value === "" ? "" : Number(event.target.value))
                    }
                  />
                );
              }
              return (
                <input
                  key={`${label}-${rowIndex}-${column.key}`}
                  value={String(cellValue ?? "")}
                  onChange={(event) => handleCellChange(event.target.value)}
                />
              );
            })}
            <button
              type="button"
              className="ghost-button"
              onClick={() => onChange(rows.filter((_, idx) => idx !== rowIndex))}
            >
              Remover
            </button>
          </div>
        ))}
        <button
          type="button"
          className="ghost-button"
          onClick={() => onChange([...rows, {}])}
        >
          Adicionar linha
        </button>
      </div>
    </div>
  );
}
