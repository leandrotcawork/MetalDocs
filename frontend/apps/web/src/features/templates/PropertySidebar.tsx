import { useState } from "react";
import { PropriedadesTab } from "./tabs/PropriedadesTab";
import { ColorPicker } from "./controls/ColorPicker";
import { SelectControl } from "./controls/SelectControl";
import { ToggleControl } from "./controls/ToggleControl";
import {
  sectionStyleFieldSchema,
  sectionCapsFieldSchema,
  dataTableStyleFieldSchema,
  dataTableCapsFieldSchema,
  repeatableStyleFieldSchema,
  repeatableCapsFieldSchema,
  repeatableItemStyleFieldSchema,
  repeatableItemCapsFieldSchema,
  richBlockStyleFieldSchema,
  richBlockCapsFieldSchema,
} from "../documents/mddm-editor/engine/codecs";

// ---------------------------------------------------------------------------
// Type helpers
// ---------------------------------------------------------------------------

type StyleSchema = readonly { key: string; label: string; type: string; default: unknown; options?: readonly string[] }[];
type CapsSchema = readonly { key: string; label: string; type: string; default: unknown; options?: readonly string[] }[];

function getSchemas(blockType: string): { style: StyleSchema; caps: CapsSchema } | null {
  switch (blockType) {
    case "section":
      return { style: sectionStyleFieldSchema, caps: sectionCapsFieldSchema };
    case "dataTable":
      return { style: dataTableStyleFieldSchema, caps: dataTableCapsFieldSchema };
    case "repeatable":
      return { style: repeatableStyleFieldSchema, caps: repeatableCapsFieldSchema };
    case "repeatableItem":
      return { style: repeatableItemStyleFieldSchema, caps: repeatableItemCapsFieldSchema };
    case "richBlock":
      return { style: richBlockStyleFieldSchema, caps: richBlockCapsFieldSchema };
    default:
      return null;
  }
}

// ---------------------------------------------------------------------------
// Schema-driven field renderer
// ---------------------------------------------------------------------------

interface SchemaFieldRendererProps {
  schema: StyleSchema | CapsSchema;
  values: Record<string, unknown>;
  onChange: (key: string, value: unknown) => void;
  testIdPrefix: string;
}

function SchemaFieldRenderer({ schema, values, onChange, testIdPrefix }: SchemaFieldRendererProps) {
  return (
    <div>
      {schema.map((field) => {
        const rawVal = values[field.key];

        if (field.type === "color") {
          const val = typeof rawVal === "string" ? rawVal : (field.default as string);
          return (
            <ColorPicker
              key={field.key}
              label={field.label}
              value={val}
              testId={`${testIdPrefix}-${field.key}`}
              onChange={(v) => onChange(field.key, v)}
            />
          );
        }

        if (field.type === "toggle") {
          const val = typeof rawVal === "boolean" ? rawVal : (field.default as boolean);
          return (
            <ToggleControl
              key={field.key}
              label={field.label}
              value={val}
              testId={`${testIdPrefix}-${field.key}`}
              onChange={(v) => onChange(field.key, v)}
            />
          );
        }

        if (field.type === "select" && (field as any).options) {
          const val = typeof rawVal === "string" ? rawVal : String(field.default);
          return (
            <SelectControl
              key={field.key}
              label={field.label}
              value={val}
              options={(field as any).options as readonly string[]}
              testId={`${testIdPrefix}-${field.key}`}
              onChange={(v) => onChange(field.key, v)}
            />
          );
        }

        if (field.type === "number") {
          const val = typeof rawVal === "number" ? rawVal : (field.default as number);
          return (
            <div key={field.key} style={{ marginBottom: "8px" }}>
              <label style={{ display: "block", fontSize: "12px", color: "#43322d", marginBottom: "4px" }}>
                {field.label}
              </label>
              <input
                data-testid={`${testIdPrefix}-${field.key}`}
                type="number"
                value={val}
                onChange={(e) => onChange(field.key, Number(e.target.value))}
                style={{
                  width: "100%",
                  background: "#ffffff",
                  border: "1px solid #c4b8b0",
                  borderRadius: "4px",
                  color: "#2b211d",
                  fontSize: "12px",
                  padding: "4px 6px",
                  boxSizing: "border-box",
                }}
              />
            </div>
          );
        }

        // "string" and unrecognized — render text input
        if (field.type === "string") {
          const val = typeof rawVal === "string" ? rawVal : (field.default as string);
          return (
            <div key={field.key} style={{ marginBottom: "8px" }}>
              <label style={{ display: "block", fontSize: "12px", color: "#43322d", marginBottom: "4px" }}>
                {field.label}
              </label>
              <input
                data-testid={`${testIdPrefix}-${field.key}`}
                type="text"
                value={val}
                onChange={(e) => onChange(field.key, e.target.value)}
                style={{
                  width: "100%",
                  background: "#ffffff",
                  border: "1px solid #c4b8b0",
                  borderRadius: "4px",
                  color: "#2b211d",
                  fontSize: "12px",
                  padding: "4px 6px",
                  boxSizing: "border-box",
                }}
              />
            </div>
          );
        }

        // Skip "string[]" and other complex types for now
        return null;
      })}
    </div>
  );
}

// ---------------------------------------------------------------------------
// EstiloTab
// ---------------------------------------------------------------------------

interface EstiloTabProps {
  block: any;
  editor: any;
  schema: StyleSchema;
}

function EstiloTab({ block, editor, schema }: EstiloTabProps) {
  const blockId: string = block.id;
  const styleJson: string = block.props?.styleJson ?? "{}";
  const currentStyle: Record<string, unknown> = (() => {
    try { return JSON.parse(styleJson); } catch { return {}; }
  })();

  const handleChange = (key: string, value: unknown) => {
    const updated = { ...currentStyle, [key]: value };
    editor.updateBlock(blockId, { props: { styleJson: JSON.stringify(updated) } });
  };

  return <SchemaFieldRenderer schema={schema} values={currentStyle} onChange={handleChange} testIdPrefix="template-style" />;
}

// ---------------------------------------------------------------------------
// CapacidadesTab
// ---------------------------------------------------------------------------

interface CapacidadesTabProps {
  block: any;
  editor: any;
  schema: CapsSchema;
}

function CapacidadesTab({ block, editor, schema }: CapacidadesTabProps) {
  const blockId: string = block.id;
  const capsJson: string = block.props?.capabilitiesJson ?? "{}";
  const currentCaps: Record<string, unknown> = (() => {
    try { return JSON.parse(capsJson); } catch { return {}; }
  })();

  const handleChange = (key: string, value: unknown) => {
    const updated = { ...currentCaps, [key]: value };
    editor.updateBlock(blockId, { props: { capabilitiesJson: JSON.stringify(updated) } });
  };

  return <SchemaFieldRenderer schema={schema} values={currentCaps} onChange={handleChange} testIdPrefix="template-caps" />;
}

// ---------------------------------------------------------------------------
// PropertySidebar
// ---------------------------------------------------------------------------

type TabId = "propriedades" | "estilo" | "capacidades";

const TABS: { id: TabId; label: string }[] = [
  { id: "propriedades", label: "Propriedades" },
  { id: "estilo", label: "Estilo" },
  { id: "capacidades", label: "Capacidades" },
];

interface Props {
  editor: any;
  selectedBlockId: string | null;
}

export function PropertySidebar({ editor, selectedBlockId }: Props) {
  const [activeTab, setActiveTab] = useState<TabId>("propriedades");

  const block = selectedBlockId && editor ? editor.getBlock(selectedBlockId) : null;

  const sidebarStyle: React.CSSProperties = {
    width: "320px",
    minWidth: "320px",
    borderLeft: "1px solid rgba(255,255,255,0.08)",
    background: "#d8d1cb",
    color: "#2b211d",
    display: "flex",
    flexDirection: "column",
    overflow: "hidden",
  };

  const tabBarStyle: React.CSSProperties = {
    display: "flex",
    borderBottom: "1px solid rgba(43,33,29,0.14)",
    flexShrink: 0,
  };

  const tabButtonStyle = (id: TabId): React.CSSProperties => ({
    flex: 1,
    padding: "8px 4px",
    fontSize: "11px",
    fontWeight: activeTab === id ? 600 : 400,
    color: activeTab === id ? "#2b211d" : "#6e605a",
    background: activeTab === id ? "rgba(255,255,255,0.45)" : "transparent",
    border: "none",
    borderBottom: activeTab === id ? "2px solid #3b82f6" : "2px solid transparent",
    cursor: "pointer",
    transition: "color 0.15s",
  });

  const contentStyle: React.CSSProperties = {
    flex: 1,
    overflowY: "auto",
    padding: "12px",
  };

  const placeholderStyle: React.CSSProperties = {
    fontSize: "12px",
    color: "#6e605a",
    textAlign: "center",
    marginTop: "32px",
    padding: "0 16px",
    lineHeight: 1.5,
  };

  if (!block) {
    return (
      <div style={sidebarStyle} data-testid="property-sidebar" data-contrast="high">
        <div style={tabBarStyle}>
          {TABS.map((tab) => (
            <button data-testid={`property-tab-${tab.id}`} key={tab.id} style={tabButtonStyle(tab.id)} onClick={() => setActiveTab(tab.id)}>
              {tab.label}
            </button>
          ))}
        </div>
        <div style={contentStyle}>
          <p style={placeholderStyle}>Selecione um bloco para editar suas propriedades.</p>
        </div>
      </div>
    );
  }

  const schemas = getSchemas(block.type);

  return (
    <div style={sidebarStyle} data-testid="property-sidebar" data-contrast="high">
      <div style={tabBarStyle}>
        {TABS.map((tab) => (
          <button data-testid={`property-tab-${tab.id}`} key={tab.id} style={tabButtonStyle(tab.id)} onClick={() => setActiveTab(tab.id)}>
            {tab.label}
          </button>
        ))}
      </div>

      {/* Block type badge */}
      <div data-testid="property-sidebar-block-type" style={{ padding: "8px 12px", borderBottom: "1px solid rgba(43,33,29,0.12)", fontSize: "11px", color: "#6e605a" }}>
        Tipo: <span style={{ color: "#2b211d", fontFamily: "monospace" }}>{block.type}</span>
      </div>

      <div style={contentStyle}>
        {activeTab === "propriedades" && (
          <PropriedadesTab block={block} editor={editor} />
        )}

        {activeTab === "estilo" && (
          schemas ? (
            <EstiloTab block={block} editor={editor} schema={schemas.style} />
          ) : (
            <p style={placeholderStyle}>Este tipo de bloco não tem campos de estilo.</p>
          )
        )}

        {activeTab === "capacidades" && (
          schemas ? (
            <CapacidadesTab block={block} editor={editor} schema={schemas.caps} />
          ) : (
            <p style={placeholderStyle}>Este tipo de bloco não tem campos de capacidades.</p>
          )
        )}
      </div>
    </div>
  );
}
