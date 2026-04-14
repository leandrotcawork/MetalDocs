interface Props {
  label: string;
  testId?: string;
  value: string;
  onChange: (value: string) => void;
}

export function ColorPicker({ label, testId, value, onChange }: Props) {
  return (
    <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: "8px" }}>
      <label style={{ fontSize: "12px", color: "rgba(255,255,255,0.7)", flex: 1 }}>{label}</label>
      <input
        data-testid={testId}
        type="color"
        value={value || "#000000"}
        onChange={(e) => onChange(e.target.value)}
        style={{ width: "32px", height: "24px", border: "none", cursor: "pointer", background: "none" }}
      />
    </div>
  );
}
