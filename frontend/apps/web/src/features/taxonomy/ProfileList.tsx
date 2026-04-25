import { useState } from "react";
import type { DocumentProfile } from "./types";
import { archiveProfile } from "./api";
import { ProfileEditDialog } from "./ProfileEditDialog";

type Props = {
  profiles: DocumentProfile[];
  includeArchived: boolean;
  onToggleArchived: (value: boolean) => void;
  onRefresh: () => void;
};

export function ProfileList({ profiles, includeArchived, onToggleArchived, onRefresh }: Props) {
  const [dialogMode, setDialogMode] = useState<"create" | "edit" | null>(null);
  const [selectedProfile, setSelectedProfile] = useState<DocumentProfile | undefined>(undefined);

  function openCreate() {
    setSelectedProfile(undefined);
    setDialogMode("create");
  }

  function openEdit(profile: DocumentProfile) {
    setSelectedProfile(profile);
    setDialogMode("edit");
  }

  function closeDialog() {
    setDialogMode(null);
    setSelectedProfile(undefined);
  }

  async function handleArchive(profile: DocumentProfile) {
    if (!window.confirm(`Arquivar perfil "${profile.name}" (${profile.code})?`)) return;
    try {
      await archiveProfile(profile.code);
      onRefresh();
    } catch (err) {
      window.alert(err instanceof Error ? err.message : "Falha ao arquivar.");
    }
  }

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 12 }}>
        <button type="button" onClick={openCreate} style={{ padding: "6px 14px" }}>
          + Novo Perfil
        </button>
        <label style={{ fontSize: 13, display: "flex", alignItems: "center", gap: 4 }}>
          <input
            type="checkbox"
            checked={includeArchived}
            onChange={(e) => onToggleArchived(e.target.checked)}
          />
          Mostrar arquivados
        </label>
      </div>

      <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
        <thead>
          <tr style={{ borderBottom: "2px solid #e0e0e0", textAlign: "left" }}>
            <th style={{ padding: "6px 8px" }}>Codigo</th>
            <th style={{ padding: "6px 8px" }}>Nome</th>
            <th style={{ padding: "6px 8px" }}>Familia</th>
            <th style={{ padding: "6px 8px" }}>Template padrao</th>
            <th style={{ padding: "6px 8px" }}>Status</th>
            <th style={{ padding: "6px 8px" }}>Acoes</th>
          </tr>
        </thead>
        <tbody>
          {profiles.length === 0 && (
            <tr>
              <td colSpan={6} style={{ padding: "12px 8px", color: "#888", textAlign: "center" }}>
                Nenhum perfil encontrado.
              </td>
            </tr>
          )}
          {profiles.map((profile) => (
            <tr
              key={profile.code}
              style={{
                borderBottom: "1px solid #f0f0f0",
                opacity: profile.archivedAt ? 0.5 : 1,
              }}
            >
              <td style={{ padding: "6px 8px", fontFamily: "monospace" }}>{profile.code}</td>
              <td style={{ padding: "6px 8px" }}>{profile.name}</td>
              <td style={{ padding: "6px 8px", fontFamily: "monospace" }}>{profile.familyCode}</td>
              <td style={{ padding: "6px 8px", fontFamily: "monospace", fontSize: 11, color: "#666" }}>
                {profile.defaultTemplateVersionId ?? "-"}
              </td>
              <td style={{ padding: "6px 8px" }}>
                {profile.archivedAt ? (
                  <span style={{ color: "#999", fontSize: 11 }}>Arquivado</span>
                ) : (
                  <span style={{ color: "#2a7a2a", fontSize: 11 }}>Ativo</span>
                )}
              </td>
              <td style={{ padding: "6px 8px", display: "flex", gap: 6 }}>
                <button type="button" onClick={() => openEdit(profile)} style={{ padding: "3px 8px", fontSize: 12 }}>
                  Editar
                </button>
                {!profile.archivedAt && (
                  <button
                    type="button"
                    onClick={() => void handleArchive(profile)}
                    style={{ padding: "3px 8px", fontSize: 12, color: "#c00" }}
                  >
                    Arquivar
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {dialogMode && (
        <ProfileEditDialog
          mode={dialogMode}
          profile={selectedProfile}
          onClose={closeDialog}
          onSaved={onRefresh}
        />
      )}
    </div>
  );
}
