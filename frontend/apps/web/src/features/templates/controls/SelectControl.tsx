interface Option {
  value: string;
  label: string;
}

interface Props {
  label: string;
  value: string;
  options: readonly string[] | readonly Option[];
  onChange: (value: string) => void;
}

function toOption(opt: string | Option): Option {
  return typeof opt === "string" ? { value: opt, label: opt } : opt;
}

export function SelectControl({ label, value, options, onChange }: Props) {
  return (
    <div style={{ marginBottom: "8px" }}>
      <label style={{ display: "block", fontSize: "12px", color: "rgba(255,255,255,0.7)", marginBottom: "4px" }}>
        {label}
      </label>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        style={{
          width: "100%",
          background: "rgba(255,255,255,0.08)",
          border: "1px solid rgba(255,255,255,0.12)",
          borderRadius: "4px",
          color: "rgba(255,255,255,0.9)",
          fontSize: "12px",
          padding: "4px 6px",
        }}
      >
        {(options as readonly (string | Option)[]).map((opt) => {
          const { value: v, label: l } = toOption(opt);
          return (
            <option key={v} value={v}>{l}</option>
          );
        })}
      </select>
    </div>
  );
}
