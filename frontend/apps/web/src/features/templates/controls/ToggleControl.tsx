interface Props {
  label: string;
  testId?: string;
  value: boolean;
  onChange: (value: boolean) => void;
}

export function ToggleControl({ label, testId, value, onChange }: Props) {
  return (
    <div style={{ display: "flex", alignItems: "center", marginBottom: "8px" }}>
      <label style={{ display: "flex", alignItems: "center", gap: "8px", fontSize: "12px", color: "rgba(255,255,255,0.7)", cursor: "pointer" }}>
        <input
          data-testid={testId}
          type="checkbox"
          checked={value}
          onChange={(e) => onChange(e.target.checked)}
          style={{ cursor: "pointer" }}
        />
        {label}
      </label>
    </div>
  );
}
