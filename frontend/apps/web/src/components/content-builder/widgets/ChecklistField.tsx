import type { ChecklistItem } from "../contentSchemaTypes";

type ChecklistFieldProps = {
  label: string;
  required?: boolean;
  value: unknown;
  onChange: (next: ChecklistItem[]) => void;
};

export function ChecklistField({ label, required, value, onChange }: ChecklistFieldProps) {
  const items = normalizeChecklistItems(value);
  return (
    <div className="content-builder-field">
      <span>
        {label}
        {required && <em className="content-builder-required">*</em>}
      </span>
      <div className="content-builder-checklist">
        {items.map((item, index) => (
          <div key={`${label}-${index}`} className="content-builder-checklist-row">
            <input
              type="checkbox"
              checked={item.checked}
              onChange={(event) => {
                const next = [...items];
                next[index] = { ...item, checked: event.target.checked };
                onChange(next);
              }}
            />
            <input
              value={item.label}
              onChange={(event) => {
                const next = [...items];
                next[index] = { ...item, label: event.target.value };
                onChange(next);
              }}
            />
            <button
              type="button"
              className="ghost-button"
              onClick={() => onChange(items.filter((_, itemIndex) => itemIndex !== index))}
            >
              Remover
            </button>
          </div>
        ))}
        <button
          type="button"
          className="ghost-button"
          onClick={() => onChange([...items, { label: "", checked: false }])}
        >
          Adicionar item
        </button>
      </div>
    </div>
  );
}

function normalizeChecklistItems(value: unknown): ChecklistItem[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.map((item) => {
    if (typeof item === "string") {
      return { label: item, checked: false };
    }
    if (item && typeof item === "object") {
      const typed = item as { label?: string; checked?: boolean };
      return { label: typed.label ?? "", checked: Boolean(typed.checked) };
    }
    return { label: "", checked: false };
  });
}
