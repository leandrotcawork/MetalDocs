import type { SchemaField } from "../contentSchemaTypes";

type PreviewTableFieldProps = {
  label: string;
  rows: Record<string, unknown>[];
  columns: SchemaField[];
};

export function PreviewTableField({ label, rows, columns }: PreviewTableFieldProps) {
  if (columns.length === 0) return null;

  const hasData = rows.length > 0 && rows.some((row) => columns.some((col) => {
    const val = row[col.key];
    return val !== undefined && val !== null && val !== "";
  }));

  return (
    <div className="preview-field preview-field-table">
      <span className="preview-field-label">{label}</span>
      <table className="preview-table">
        <thead>
          <tr>
            {columns.map((col) => (
              <th key={col.key}>{col.label ?? col.key}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {hasData ? (
            rows.map((row, i) => (
              <tr key={i}>
                {columns.map((col) => (
                  <td key={col.key}>{String(row[col.key] ?? "")}</td>
                ))}
              </tr>
            ))
          ) : (
            <tr>
              <td colSpan={columns.length} className="preview-table-empty">
                —
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
