type PreviewTextFieldProps = {
  label: string;
  value: string;
};

export function PreviewTextField({ label, value }: PreviewTextFieldProps) {
  if (!value) {
    return (
      <div className="preview-field preview-field-text is-empty">
        <span className="preview-field-label">{label}</span>
        <span className="preview-field-placeholder">—</span>
      </div>
    );
  }

  return (
    <div className="preview-field preview-field-text">
      <span className="preview-field-label">{label}</span>
      <p className="preview-field-value">{value}</p>
    </div>
  );
}
