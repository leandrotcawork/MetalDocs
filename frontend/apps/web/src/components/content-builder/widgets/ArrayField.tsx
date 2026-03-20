type ArrayFieldProps = {
  label: string;
  required?: boolean;
  items: string[];
  onChange: (next: string[]) => void;
};

export function ArrayField({ label, required, items, onChange }: ArrayFieldProps) {
  return (
    <div className="content-builder-field">
      <span>
        {label}
        {required && <em className="content-builder-required">*</em>}
      </span>
      <div className="content-builder-array">
        {items.map((item, index) => (
          <div key={`${label}-${index}`} className="content-builder-array-row">
            <input
              value={item}
              onChange={(event) => {
                const next = [...items];
                next[index] = event.target.value;
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
          onClick={() => onChange([...items, ""])}
        >
          Adicionar item
        </button>
      </div>
    </div>
  );
}
