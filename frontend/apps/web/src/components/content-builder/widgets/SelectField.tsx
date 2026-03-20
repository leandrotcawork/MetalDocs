type SelectFieldProps = {
  label: string;
  required?: boolean;
  value: string;
  options: string[];
  onChange: (next: string) => void;
};

export function SelectField({ label, required, value, options, onChange }: SelectFieldProps) {
  return (
    <label className="content-builder-field">
      <span>
        {label}
        {required && <em className="content-builder-required">*</em>}
      </span>
      <select value={value} onChange={(event) => onChange(event.target.value)}>
        <option value="">Selecione</option>
        {options.map((option) => (
          <option key={option} value={option}>{option}</option>
        ))}
      </select>
    </label>
  );
}
