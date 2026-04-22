// TODO: This replaces RegistryExplorerView.tsx — switch routing in Phase 8 or 9
import { useEffect, useState } from "react";
import { fetchControlledDocuments } from "./api";
import type { ControlledDocument } from "./types";
import { RegistryCreateDialog } from "./RegistryCreateDialog";
import { RegistryDetailPage } from "./RegistryDetailPage";

export function RegistryListPage() {
  const [docs, setDocs] = useState<ControlledDocument[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [profileFilter, setProfileFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [detailId, setDetailId] = useState<string | null>(null);

  async function load() {
    setLoading(true);
    setError("");
    try {
      const filter: Parameters<typeof fetchControlledDocuments>[0] = {};
      if (profileFilter) filter.profileCode = profileFilter;
      if (statusFilter) filter.status = statusFilter;
      const rows = await fetchControlledDocuments(filter);
      setDocs(rows);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load.");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [profileFilter, statusFilter]);

  const profiles = Array.from(new Set(docs.map((d) => d.profileCode))).sort();

  if (detailId) {
    return (
      <RegistryDetailPage
        id={detailId}
        onBack={() => {
          setDetailId(null);
          void load();
        }}
      />
    );
  }

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 16 }}>
        <h2 style={{ margin: 0, fontSize: 16 }}>Documentos Controlados</h2>
        <button type="button" onClick={() => setShowCreate(true)} style={{ marginLeft: "auto", padding: "6px 14px" }}>
          + Novo
        </button>
      </div>

      <div style={{ display: "flex", gap: 12, marginBottom: 14 }}>
        <div>
          <label style={{ fontSize: 12, marginRight: 6 }}>Perfil:</label>
          <select value={profileFilter} onChange={(e) => setProfileFilter(e.target.value)} style={{ padding: "4px 8px" }}>
            <option value="">Todos</option>
            {profiles.map((p) => <option key={p} value={p}>{p}</option>)}
          </select>
        </div>
        <div>
          <label style={{ fontSize: 12, marginRight: 6 }}>Status:</label>
          <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} style={{ padding: "4px 8px" }}>
            <option value="">Todos</option>
            <option value="active">Ativo</option>
            <option value="obsolete">Obsoleto</option>
            <option value="superseded">Supersedido</option>
          </select>
        </div>
      </div>

      {loading && <div style={{ color: "#888", fontSize: 13 }}>Carregando...</div>}
      {error && <div style={{ color: "#c00", fontSize: 13 }}>{error}</div>}

      {!loading && !error && (
        <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
          <thead>
            <tr style={{ borderBottom: "2px solid #e0e0e0", textAlign: "left" }}>
              <th style={{ padding: "6px 8px" }}>Codigo</th>
              <th style={{ padding: "6px 8px" }}>Titulo</th>
              <th style={{ padding: "6px 8px" }}>Perfil</th>
              <th style={{ padding: "6px 8px" }}>Area</th>
              <th style={{ padding: "6px 8px" }}>Status</th>
            </tr>
          </thead>
          <tbody>
            {docs.length === 0 && (
              <tr>
                <td colSpan={5} style={{ padding: "12px 8px", color: "#888", textAlign: "center" }}>
                  Nenhum documento controlado encontrado.
                </td>
              </tr>
            )}
            {docs.map((doc) => (
              <tr
                key={doc.id}
                onClick={() => setDetailId(doc.id)}
                style={{ borderBottom: "1px solid #f0f0f0", cursor: "pointer" }}
                onMouseOver={(e) => { (e.currentTarget as HTMLTableRowElement).style.background = "#f7f7f7"; }}
                onMouseOut={(e) => { (e.currentTarget as HTMLTableRowElement).style.background = ""; }}
              >
                <td style={{ padding: "6px 8px", fontFamily: "monospace" }}>{doc.code}</td>
                <td style={{ padding: "6px 8px" }}>{doc.title}</td>
                <td style={{ padding: "6px 8px", fontFamily: "monospace" }}>{doc.profileCode}</td>
                <td style={{ padding: "6px 8px", fontFamily: "monospace" }}>{doc.processAreaCode}</td>
                <td style={{ padding: "6px 8px", fontSize: 11, fontWeight: 600, color: doc.status === "active" ? "#2a7a2a" : doc.status === "obsolete" ? "#888" : "#b87a00" }}>
                  {doc.status.toUpperCase()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {showCreate && (
        <RegistryCreateDialog
          onClose={() => setShowCreate(false)}
          onCreated={() => {
            setShowCreate(false);
            void load();
          }}
        />
      )}
    </div>
  );
}
