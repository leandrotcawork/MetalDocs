type ChecklistItem = {
  label: string;
  checked: boolean;
};

type PreviewChecklistFieldProps = {
  label: string;
  value: unknown;
};

export function PreviewChecklistField({ label, value }: PreviewChecklistFieldProps) {
  const items: ChecklistItem[] = Array.isArray(value)
    ? (value as ChecklistItem[]).filter((item) => item && typeof item.label === "string")
    : [];

  return (
    <div className="preview-field preview-field-checklist">
      <span className="preview-field-label">{label}</span>
      {items.length > 0 ? (
        <ul className="preview-checklist">
          {items.map((item, i) => (
            <li key={i} className={item.checked ? "is-checked" : ""}>
              <span className="preview-checklist-icon">{item.checked ? "\u2611" : "\u2610"}</span>
              <span>{item.label}</span>
            </li>
          ))}
        </ul>
      ) : (
        <span className="preview-field-placeholder">—</span>
      )}
    </div>
  );
}
