type PreviewArrayFieldProps = {
  label: string;
  items: string[];
};

export function PreviewArrayField({ label, items }: PreviewArrayFieldProps) {
  const hasItems = items.length > 0 && items.some((item) => item.trim() !== "");

  return (
    <div className="preview-field preview-field-array">
      <span className="preview-field-label">{label}</span>
      {hasItems ? (
        <ul className="preview-list">
          {items.filter((item) => item.trim() !== "").map((item, i) => (
            <li key={i}>{item}</li>
          ))}
        </ul>
      ) : (
        <span className="preview-field-placeholder">—</span>
      )}
    </div>
  );
}
