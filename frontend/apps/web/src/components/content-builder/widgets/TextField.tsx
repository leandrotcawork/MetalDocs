type TextFieldProps = {
  label: string;
  required?: boolean;
  value: string;
  onChange: (next: string) => void;
  type?: string;
};

export function TextField({ label, required, value, onChange, type = "text" }: TextFieldProps) {
  return (
    <label className="content-builder-field">
      <span>
        {label}
        {required && <em className="content-builder-required">*</em>}
      </span>
      <input
        type={type}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
    </label>
  );
}
