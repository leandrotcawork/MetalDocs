const inputStyle: React.CSSProperties = {
  width: "100%",
  background: "rgba(255,255,255,0.08)",
  border: "1px solid rgba(255,255,255,0.12)",
  borderRadius: "4px",
  color: "rgba(255,255,255,0.9)",
  fontSize: "12px",
  padding: "4px 6px",
  boxSizing: "border-box",
};

const labelStyle: React.CSSProperties = {
  display: "block",
  fontSize: "12px",
  color: "rgba(255,255,255,0.7)",
  marginBottom: "4px",
};

const fieldStyle: React.CSSProperties = { marginBottom: "12px" };

interface FieldRowProps {
  label: string;
  testId?: string;
  value: string | number;
  type?: "text" | "number";
  onChange: (val: string) => void;
}

function FieldRow({ label, testId, value, type = "text", onChange }: FieldRowProps) {
  return (
    <div style={fieldStyle}>
      <label style={labelStyle}>{label}</label>
      <input
        data-testid={testId}
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        style={inputStyle}
      />
    </div>
  );
}

interface Props {
  block: any;
  editor: any;
}

export function PropriedadesTab({ block, editor }: Props) {
  const blockId: string = block.id;
  const props = block.props ?? {};

  const update = (patch: Record<string, unknown>) => {
    editor.updateBlock(blockId, { props: patch });
  };

  switch (block.type) {
    case "section":
      return (
        <div>
          <FieldRow
            label="Título da seção"
            testId="template-prop-title"
            value={props.title ?? ""}
            onChange={(val) => update({ title: val })}
          />
        </div>
      );

    case "field":
      return (
        <div>
          <FieldRow
            label="Rótulo"
            testId="template-prop-label"
            value={props.label ?? ""}
            onChange={(val) => update({ label: val })}
          />
        </div>
      );

    case "dataTable": {
      const columns: string[] = (() => {
        try {
          const raw = props.columnsJson ? JSON.parse(props.columnsJson) : [];
          return Array.isArray(raw) ? raw.map((c: any) => (typeof c === "string" ? c : c.key ?? String(c))) : [];
        } catch {
          return [];
        }
      })();
      return (
        <div>
          <p style={{ fontSize: "12px", color: "rgba(255,255,255,0.6)", marginBottom: "8px" }}>
            Colunas ({columns.length}):
          </p>
          {columns.length === 0 ? (
            <p style={{ fontSize: "11px", color: "rgba(255,255,255,0.4)" }}>Nenhuma coluna definida.</p>
          ) : (
            <ul style={{ margin: 0, paddingLeft: "1.25rem", fontSize: "12px", color: "rgba(255,255,255,0.7)" }}>
              {columns.map((col, i) => (
                <li key={i}>{col}</li>
              ))}
            </ul>
          )}
        </div>
      );
    }

    case "repeatable": {
      const caps = (() => {
        try {
          return props.capabilitiesJson ? JSON.parse(props.capabilitiesJson) : {};
        } catch {
          return {};
        }
      })();
      return (
        <div>
          <FieldRow
            label="Mín. itens"
            testId="template-prop-minItems"
            value={caps.minItems ?? 0}
            type="number"
            onChange={(val) => update({ capabilitiesJson: JSON.stringify({ ...caps, minItems: Number(val) }) })}
          />
          <FieldRow
            label="Máx. itens"
            testId="template-prop-maxItems"
            value={caps.maxItems ?? 100}
            type="number"
            onChange={(val) => update({ capabilitiesJson: JSON.stringify({ ...caps, maxItems: Number(val) }) })}
          />
        </div>
      );
    }

    case "richBlock":
      return (
        <div>
          <FieldRow
            label="Rótulo"
            testId="template-prop-label"
            value={props.label ?? ""}
            onChange={(val) => update({ label: val })}
          />
        </div>
      );

    case "repeatableItem":
      return (
        <div>
          <p style={{ fontSize: "12px", color: "rgba(255,255,255,0.5)", fontStyle: "italic" }}>
            Número do item determinado pelo bloco pai.
          </p>
        </div>
      );

    default:
      return (
        <div>
          <p style={{ fontSize: "12px", color: "rgba(255,255,255,0.5)" }}>
            Tipo de bloco não configurável.
          </p>
        </div>
      );
  }
}
