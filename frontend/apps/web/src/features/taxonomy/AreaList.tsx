import { useState } from "react";
import type { ProcessArea } from "./types";
import { archiveArea } from "./api";
import { AreaEditDialog } from "./AreaEditDialog";

type Props = {
  areas: ProcessArea[];
  includeArchived: boolean;
  onToggleArchived: (value: boolean) => void;
  onRefresh: () => void;
};

export function AreaList({ areas, includeArchived, onToggleArchived, onRefresh }: Props) {
  const [dialogMode, setDialogMode] = useState<"create" | "edit" | null>(null);
  const [selectedArea, setSelectedArea] = useState<ProcessArea | undefined>(undefined);

  function openCreate() {
    setSelectedArea(undefined);
    setDialogMode("create");
  }

  function openEdit(area: ProcessArea) {
    setSelectedArea(area);
    setDialogMode("edit");
  }

  function closeDialog() {
    setDialogMode(null);
    setSelectedArea(undefined);
  }

  async function handleArchive(area: ProcessArea) {
    if (!window.confirm(`Arquivar area "${area.name}" (${area.code})?`)) return;
    try {
      await archiveArea(area.code);
      onRefresh();
    } catch (err) {
      window.alert(err instanceof Error ? err.message : "Falha ao arquivar.");
    }
  }

  const areaByCode = new Map(areas.map((a) => [a.code, a]));

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 12 }}>
        <button type="button" onClick={openCreate} style={{ padding: "6px 14px" }}>
          + Nova Area
        </button>
        <label style={{ fontSize: 13, display: "flex", alignItems: "center", gap: 4 }}>
          <input
            type="checkbox"
            checked={includeArchived}
            onChange={(e) => onToggleArchived(e.target.checked)}
          />
          Mostrar arquivadas
        </label>
      </div>

      <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
        <thead>
          <tr style={{ borderBottom: "2px solid #e0e0e0", textAlign: "left" }}>
            <th style={{ padding: "6px 8px" }}>Codigo</th>
            <th style={{ padding: "6px 8px" }}>Nome</th>
            <th style={{ padding: "6px 8px" }}>Area pai</th>
            <th style={{ padding: "6px 8px" }}>Role aprovador</th>
            <th style={{ padding: "6px 8px" }}>Status</th>
            <th style={{ padding: "6px 8px" }}>Acoes</th>
          </tr>
        </thead>
        <tbody>
          {areas.length === 0 && (
            <tr>
              <td colSpan={6} style={{ padding: "12px 8px", color: "#888", textAlign: "center" }}>
                Nenhuma area encontrada.
              </td>
            </tr>
          )}
          {areas.map((area) => (
            <tr
              key={area.code}
              style={{
                borderBottom: "1px solid #f0f0f0",
                opacity: area.archivedAt ? 0.5 : 1,
              }}
            >
              <td style={{ padding: "6px 8px", fontFamily: "monospace" }}>{area.code}</td>
              <td style={{ padding: "6px 8px" }}>{area.name}</td>
              <td style={{ padding: "6px 8px", fontFamily: "monospace", fontSize: 12 }}>
                {area.parentCode ? (areaByCode.get(area.parentCode)?.name ?? area.parentCode) : "-"}
              </td>
              <td style={{ padding: "6px 8px", fontSize: 12 }}>{area.defaultApproverRole ?? "-"}</td>
              <td style={{ padding: "6px 8px" }}>
                {area.archivedAt ? (
                  <span style={{ color: "#999", fontSize: 11 }}>Arquivada</span>
                ) : (
                  <span style={{ color: "#2a7a2a", fontSize: 11 }}>Ativa</span>
                )}
              </td>
              <td style={{ padding: "6px 8px", display: "flex", gap: 6 }}>
                <button type="button" onClick={() => openEdit(area)} style={{ padding: "3px 8px", fontSize: 12 }}>
                  Editar
                </button>
                {!area.archivedAt && (
                  <button
                    type="button"
                    onClick={() => void handleArchive(area)}
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
        <AreaEditDialog
          mode={dialogMode}
          area={selectedArea}
          areas={areas}
          onClose={closeDialog}
          onSaved={onRefresh}
        />
      )}
    </div>
  );
}
