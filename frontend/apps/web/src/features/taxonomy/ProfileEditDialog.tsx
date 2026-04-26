import { useState } from "react";
import type { DocumentProfile, CreateProfileRequest, UpdateProfileRequest } from "./types";
import { createProfile, updateProfile, setDefaultTemplate } from "./api";

type Props = {
  mode: "create" | "edit";
  profile?: DocumentProfile;
  onClose: () => void;
  onSaved: () => void;
};

export function ProfileEditDialog({ mode, profile, onClose, onSaved }: Props) {
  const [code, setCode] = useState(profile?.code ?? "");
  const [familyCode, setFamilyCode] = useState(profile?.familyCode ?? "");
  const [name, setName] = useState(profile?.name ?? "");
  const [description, setDescription] = useState(profile?.description ?? "");
  const [reviewIntervalDays, setReviewIntervalDays] = useState(String(profile?.reviewIntervalDays ?? 365));
  const [editableByRole, setEditableByRole] = useState(profile?.editableByRole ?? "admin");
  const [templateVersionId, setTemplateVersionId] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [templateError, setTemplateError] = useState("");
  const [templateSaving, setTemplateSaving] = useState(false);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSaving(true);
    try {
      if (mode === "create") {
        const req: CreateProfileRequest = {
          code: code.trim(),
          familyCode: familyCode.trim(),
          name: name.trim(),
          description: description.trim() || undefined,
          reviewIntervalDays: Number(reviewIntervalDays),
          editableByRole: editableByRole.trim() || undefined,
        };
        await createProfile(req);
      } else {
        const req: UpdateProfileRequest = {
          familyCode: familyCode.trim(),
          name: name.trim(),
          description: description.trim() || undefined,
          editableByRole: editableByRole.trim() || undefined,
          reviewIntervalDays: Number(reviewIntervalDays),
        };
        await updateProfile(profile!.code, req);
      }
      onSaved();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao salvar.");
    } finally {
      setSaving(false);
    }
  }

  async function handleSetTemplate(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setTemplateError("");
    setTemplateSaving(true);
    try {
      await setDefaultTemplate(profile!.code, { templateVersionId: templateVersionId.trim() });
      setTemplateVersionId("");
    } catch (err) {
      setTemplateError(err instanceof Error ? err.message : "Falha ao definir template.");
    } finally {
      setTemplateSaving(false);
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
          {mode === "create" ? "Novo Perfil Documental" : "Editar Perfil Documental"}
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
              <input value={profile?.code ?? ""} readOnly style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box", background: "#f5f5f5" }} />
            </div>
          )}
          <div style={{ marginBottom: 12 }}>
            <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>Familia *</label>
            <input
              value={familyCode}
              onChange={(e) => setFamilyCode(e.target.value)}
              required={mode === "create"}
              readOnly={mode === "edit"}
              style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box", background: mode === "edit" ? "#f5f5f5" : undefined }}
            />
          </div>
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
            <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>Intervalo de revisao (dias) *</label>
            <input
              type="number"
              min={1}
              value={reviewIntervalDays}
              onChange={(e) => setReviewIntervalDays(e.target.value)}
              required
              style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box" }}
            />
          </div>
          <div style={{ marginBottom: 16 }}>
            <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>Role editora</label>
            <input
              value={editableByRole}
              onChange={(e) => setEditableByRole(e.target.value)}
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

        {mode === "edit" && (
          <div style={{ marginTop: 24, borderTop: "1px solid #eee", paddingTop: 16 }}>
            <h3 style={{ margin: "0 0 12px", fontSize: 13 }}>Template padrao</h3>
            <form onSubmit={(e) => void handleSetTemplate(e)}>
              <div style={{ marginBottom: 8 }}>
                <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>ID da versao do template (UUID)</label>
                <input
                  value={templateVersionId}
                  onChange={(e) => setTemplateVersionId(e.target.value)}
                  placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                  required
                  style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box" }}
                />
              </div>
              {templateError && <p style={{ color: "#c00", fontSize: 12, marginBottom: 8 }}>{templateError}</p>}
              <button type="submit" disabled={templateSaving} style={{ padding: "6px 14px" }}>
                {templateSaving ? "Definindo..." : "Definir template padrao"}
              </button>
            </form>
          </div>
        )}
      </div>
    </div>
  );
}
