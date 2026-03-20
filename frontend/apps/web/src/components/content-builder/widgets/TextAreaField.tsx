type TextAreaFieldProps = {
  label: string;
  required?: boolean;
  value: string;
  onChange: (next: string) => void;
  rows?: number;
};

export function TextAreaField({ label, required, value, onChange, rows = 4 }: TextAreaFieldProps) {
  return (
    <label className="content-builder-field">
      <span>
        {label}
        {required && <em className="content-builder-required">*</em>}
      </span>
      <textarea
        value={value}
        onChange={(event) => onChange(event.target.value)}
        rows={rows}
      />
    </label>
  );
}
