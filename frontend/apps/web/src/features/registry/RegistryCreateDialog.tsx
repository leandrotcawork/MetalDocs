import { useEffect, useState } from "react";
import type { DocumentProfile, ProcessArea } from "../taxonomy/types";
import { fetchProfiles, fetchAreas } from "../taxonomy/api";
import { createControlledDocument } from "./api";
import type { ControlledDocument, CreateControlledDocumentRequest } from "./types";

type Props = {
  onClose: () => void;
  onCreated: (doc: ControlledDocument) => void;
};

export function RegistryCreateDialog({ onClose, onCreated }: Props) {
  const [profiles, setProfiles] = useState<DocumentProfile[]>([]);
  const [areas, setAreas] = useState<ProcessArea[]>([]);
  const [profileCode, setProfileCode] = useState("");
  const [processAreaCode, setProcessAreaCode] = useState("");
  const [title, setTitle] = useState("");
  const [ownerUserId, setOwnerUserId] = useState("");
  const [manualCodeEnabled, setManualCodeEnabled] = useState(false);
  const [manualCode, setManualCode] = useState("");
  const [manualCodeReason, setManualCodeReason] = useState("");
  const [overrideTemplateEnabled, setOverrideTemplateEnabled] = useState(false);
  const [overrideTemplateVersionId, setOverrideTemplateVersionId] = useState("");
  const [overrideTemplateReason, setOverrideTemplateReason] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    void (async () => {
      try {
        const [p, a] = await Promise.all([fetchProfiles(), fetchAreas()]);
        setProfiles(p);
        setAreas(a);
      } catch {
        setError("Failed to load profiles/areas.");
      }
    })();
  }, []);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSaving(true);
    try {
      const req: CreateControlledDocumentRequest = {
        profileCode,
        processAreaCode,
        title: title.trim(),
        ownerUserId: ownerUserId.trim(),
      };
      if (manualCodeEnabled && manualCode.trim()) {
        req.manualCode = manualCode.trim();
        req.manualCodeReason = manualCodeReason.trim() || undefined;
      }
      if (overrideTemplateEnabled && overrideTemplateVersionId.trim()) {
        req.overrideTemplateVersionId = overrideTemplateVersionId.trim();
        req.overrideTemplateReason = overrideTemplateReason.trim() || undefined;
      }
      const doc = await createControlledDocument(req);
      onCreated(doc);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create.");
    } finally {
      setSaving(false);
    }
  }

  const autoCodePreview = profileCode ? `${profileCode.toUpperCase()}-??` : "??";

  return (
    <div
      style={{
        position: "fixed", inset: 0, background: "rgba(0,0,0,0.4)", zIndex: 1000,
        display: "flex", alignItems: "center", justifyContent: "center",
      }}
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div style={{ background: "#fff", borderRadius: 8, padding: 24, minWidth: 420, maxWidth: 540, width: "100%", maxHeight: "90vh", overflowY: "auto" }}>
        <h2 style={{ margin: "0 0 16px", fontSize: 16 }}>Novo Documento Controlado</h2>
        <form onSubmit={(e) => void handleSubmit(e)}>
          <div style={{ marginBottom: 12 }}>
            <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>Perfil *</label>
            <select
              value={profileCode}
              onChange={(e) => setProfileCode(e.target.value)}
              required
              style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box" }}
            >
              <option value="">-- Selecionar --</option>
              {profiles.map((p) => (
                <option key={p.code} value={p.code}>{p.code} — {p.name}</option>
              ))}
            </select>
          </div>

          <div style={{ marginBottom: 12 }}>
            <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>Area de processo *</label>
            <select
              value={processAreaCode}
              onChange={(e) => setProcessAreaCode(e.target.value)}
              required
              style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box" }}
            >
              <option value="">-- Selecionar --</option>
              {areas.map((a) => (
                <option key={a.code} value={a.code}>{a.code} — {a.name}</option>
              ))}
            </select>
          </div>

          <div style={{ marginBottom: 12 }}>
            <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>Titulo *</label>
            <input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              required
              style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box" }}
            />
          </div>

          <div style={{ marginBottom: 12 }}>
            <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>Dono (User ID) *</label>
            <input
              value={ownerUserId}
              onChange={(e) => setOwnerUserId(e.target.value)}
              required
              style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box" }}
            />
          </div>

          <div style={{ marginBottom: 12, padding: "8px 10px", background: "#f5f5f5", borderRadius: 4, fontSize: 12 }}>
            Codigo previsto: <strong style={{ fontFamily: "monospace" }}>{autoCodePreview}</strong>
          </div>

          <div style={{ marginBottom: 12 }}>
            <label style={{ display: "flex", alignItems: "center", gap: 6, fontSize: 12, cursor: "pointer" }}>
              <input
                type="checkbox"
                checked={manualCodeEnabled}
                onChange={(e) => setManualCodeEnabled(e.target.checked)}
              />
              Codigo manual
            </label>
            {manualCodeEnabled && (
              <div style={{ marginTop: 8 }}>
                <input
                  value={manualCode}
                  onChange={(e) => setManualCode(e.target.value)}
                  placeholder="Codigo manual"
                  style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box", marginBottom: 6 }}
                />
                <textarea
                  value={manualCodeReason}
                  onChange={(e) => setManualCodeReason(e.target.value)}
                  placeholder="Motivo (opcional)"
                  rows={2}
                  style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box" }}
                />
              </div>
            )}
          </div>

          <div style={{ marginBottom: 16 }}>
            <label style={{ display: "flex", alignItems: "center", gap: 6, fontSize: 12, cursor: "pointer" }}>
              <input
                type="checkbox"
                checked={overrideTemplateEnabled}
                onChange={(e) => setOverrideTemplateEnabled(e.target.checked)}
              />
              Override de template
            </label>
            {overrideTemplateEnabled && (
              <div style={{ marginTop: 8 }}>
                <input
                  value={overrideTemplateVersionId}
                  onChange={(e) => setOverrideTemplateVersionId(e.target.value)}
                  placeholder="Template version ID (UUID)"
                  style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box", marginBottom: 6 }}
                />
                <textarea
                  value={overrideTemplateReason}
                  onChange={(e) => setOverrideTemplateReason(e.target.value)}
                  placeholder="Motivo (opcional)"
                  rows={2}
                  style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box" }}
                />
              </div>
            )}
          </div>

          {error && <p style={{ color: "#c00", fontSize: 12, marginBottom: 8 }}>{error}</p>}
          <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
            <button type="button" onClick={onClose} style={{ padding: "6px 14px" }}>Cancelar</button>
            <button type="submit" disabled={saving} style={{ padding: "6px 14px" }}>
              {saving ? "Criando..." : "Criar"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
