interface Props {
  label: string;
  value: string;
  onChange: (value: string) => void;
}

const UNITS = ["px", "pt", "mm", "rem", "em", "%"] as const;

function parseValue(val: string): { num: string; unit: string } {
  const match = val.match(/^([\d.]*)(.*)$/);
  if (!match) return { num: "", unit: "px" };
  const num = match[1] ?? "";
  const unit = match[2]?.trim() || "px";
  return { num, unit };
}

export function NumberUnitInput({ label, value, onChange }: Props) {
  const { num, unit } = parseValue(value);

  const handleNumChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    onChange(e.target.value + unit);
  };

  const handleUnitChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    onChange(num + e.target.value);
  };

  return (
    <div style={{ marginBottom: "8px" }}>
      <label style={{ display: "block", fontSize: "12px", color: "rgba(255,255,255,0.7)", marginBottom: "4px" }}>
        {label}
      </label>
      <div style={{ display: "flex", gap: "4px" }}>
        <input
          type="number"
          value={num}
          onChange={handleNumChange}
          style={{
            flex: 1,
            background: "rgba(255,255,255,0.08)",
            border: "1px solid rgba(255,255,255,0.12)",
            borderRadius: "4px",
            color: "rgba(255,255,255,0.9)",
            fontSize: "12px",
            padding: "3px 6px",
          }}
        />
        <select
          value={UNITS.includes(unit as any) ? unit : "px"}
          onChange={handleUnitChange}
          style={{
            background: "rgba(255,255,255,0.08)",
            border: "1px solid rgba(255,255,255,0.12)",
            borderRadius: "4px",
            color: "rgba(255,255,255,0.9)",
            fontSize: "12px",
            padding: "3px 4px",
          }}
        >
          {UNITS.map((u) => (
            <option key={u} value={u}>{u}</option>
          ))}
        </select>
      </div>
    </div>
  );
}
