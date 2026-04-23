import { useState } from "react";
import type { ProcessArea, CreateAreaRequest, UpdateAreaRequest } from "./types";
import { createArea, updateArea } from "./api";

type Props = {
  mode: "create" | "edit";
  area?: ProcessArea;
  areas: ProcessArea[];
  onClose: () => void;
  onSaved: () => void;
};

export function AreaEditDialog({ mode, area, areas, onClose, onSaved }: Props) {
  const [code, setCode] = useState(area?.code ?? "");
  const [name, setName] = useState(area?.name ?? "");
  const [description, setDescription] = useState(area?.description ?? "");
  const [parentCode, setParentCode] = useState(area?.parentCode ?? "");
  const [defaultApproverRole, setDefaultApproverRole] = useState(area?.defaultApproverRole ?? "");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  const parentOptions = areas.filter((a) => a.code !== area?.code && !a.archivedAt);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSaving(true);
    try {
      if (mode === "create") {
        const req: CreateAreaRequest = {
          code: code.trim(),
          name: name.trim(),
          description: description.trim() || undefined,
          parentCode: parentCode.trim() || undefined,
          defaultApproverRole: defaultApproverRole.trim() || undefined,
        };
        await createArea(req);
      } else {
        const req: UpdateAreaRequest = {
          name: name.trim(),
          description: description.trim() || undefined,
          parentCode: parentCode.trim() || null,
          defaultApproverRole: defaultApproverRole.trim() || null,
        };
        await updateArea(area!.code, req);
      }
      onSaved();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao salvar.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div
      style={{
        position: "fixed", inset: 0, background: "rgba(0,0,0,0.4)", zIndex: 1000,
        display: "flex", alignItems: "center", justifyContent: "center",
      }}
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div style={{ background: "#fff", borderRadius: 8, padding: 24, minWidth: 400, maxWidth: 520, width: "100%" }}>
        <h2 style={{ margin: "0 0 16px", fontSize: 16 }}>
          {mode === "create" ? "Nova Area de Processo" : "Editar Area de Processo"}
        </h2>
        <form onSubmit={(e) => void handleSubmit(e)}>
          {mode === "create" && (
            <div style={{ marginBottom: 12 }}>
              <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>Codigo *</label>
              <input
                value={code}
                onChange={(e) => setCode(e.target.value)}
                required
                style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box" }}
              />
            </div>
          )}
          {mode === "edit" && (
            <div style={{ marginBottom: 12 }}>
              <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>Codigo</label>
              <input value={area?.code ?? ""} readOnly style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box", background: "#f5f5f5" }} />
            </div>
          )}
          <div style={{ marginBottom: 12 }}>
            <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>Nome *</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box" }}
            />
          </div>
          <div style={{ marginBottom: 12 }}>
            <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>Descricao</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={2}
              style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box" }}
            />
          </div>
          <div style={{ marginBottom: 12 }}>
            <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>Area pai</label>
            <select
              value={parentCode}
              onChange={(e) => setParentCode(e.target.value)}
              style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box" }}
            >
              <option value="">— Nenhuma —</option>
              {parentOptions.map((a) => (
                <option key={a.code} value={a.code}>{a.name} ({a.code})</option>
              ))}
            </select>
          </div>
          <div style={{ marginBottom: 16 }}>
            <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>Role de aprovador padrao</label>
            <input
              value={defaultApproverRole}
              onChange={(e) => setDefaultApproverRole(e.target.value)}
              style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box" }}
            />
          </div>
          {error && <p style={{ color: "#c00", fontSize: 12, marginBottom: 8 }}>{error}</p>}
          <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
            <button type="button" onClick={onClose} style={{ padding: "6px 14px" }}>Cancelar</button>
            <button type="submit" disabled={saving} style={{ padding: "6px 14px" }}>
              {saving ? "Salvando..." : "Salvar"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
