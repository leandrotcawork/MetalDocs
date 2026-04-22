import { useState } from "react";
import { fetchMemberships, revokeMembership } from "./membershipApi";
import type { AreaMembership } from "./membershipApi";
import { MembershipGrantDialog } from "./MembershipGrantDialog";

export function AreaMembershipAdminPage() {
  const [userIdInput, setUserIdInput] = useState("");
  const [memberships, setMemberships] = useState<AreaMembership[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [showGrant, setShowGrant] = useState(false);
  const [lastQuery, setLastQuery] = useState("");

  async function handleFilter(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const uid = userIdInput.trim();
    if (!uid) return;
    setLastQuery(uid);
    setLoading(true);
    setError("");
    try {
      const rows = await fetchMemberships(uid);
      setMemberships(rows);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load.");
    } finally {
      setLoading(false);
    }
  }

  async function handleRevoke(userId: string, areaCode: string) {
    if (!window.confirm(`Revogar acesso do usuario "${userId}" na area "${areaCode}"?`)) return;
    try {
      await revokeMembership(userId, areaCode);
      if (lastQuery) {
        const rows = await fetchMemberships(lastQuery);
        setMemberships(rows);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to revoke.");
    }
  }

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 16 }}>
        <h2 style={{ margin: 0, fontSize: 16 }}>Memberships de Area</h2>
        <button type="button" onClick={() => setShowGrant(true)} style={{ marginLeft: "auto", padding: "6px 14px" }}>
          + Conceder acesso
        </button>
      </div>

      <form onSubmit={(e) => void handleFilter(e)} style={{ display: "flex", gap: 8, marginBottom: 16 }}>
        <input
          value={userIdInput}
          onChange={(e) => setUserIdInput(e.target.value)}
          placeholder="Filtrar por User ID"
          style={{ padding: "6px 8px", minWidth: 240 }}
        />
        <button type="submit" style={{ padding: "6px 14px" }}>Buscar</button>
      </form>

      {loading && <div style={{ color: "#888", fontSize: 13 }}>Carregando...</div>}
      {error && <div style={{ color: "#c00", fontSize: 13 }}>{error}</div>}

      {!loading && memberships.length > 0 && (
        <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
          <thead>
            <tr style={{ borderBottom: "2px solid #e0e0e0", textAlign: "left" }}>
              <th style={{ padding: "6px 8px" }}>User ID</th>
              <th style={{ padding: "6px 8px" }}>Area</th>
              <th style={{ padding: "6px 8px" }}>Papel</th>
              <th style={{ padding: "6px 8px" }}>Desde</th>
              <th style={{ padding: "6px 8px" }}>Ate</th>
              <th style={{ padding: "6px 8px" }}>Acoes</th>
            </tr>
          </thead>
          <tbody>
            {memberships.map((m) => (
              <tr key={`${m.userId}-${m.areaCode}`} style={{ borderBottom: "1px solid #f0f0f0" }}>
                <td style={{ padding: "6px 8px", fontFamily: "monospace", fontSize: 11 }}>{m.userId}</td>
                <td style={{ padding: "6px 8px", fontFamily: "monospace" }}>{m.areaCode}</td>
                <td style={{ padding: "6px 8px" }}>{m.role}</td>
                <td style={{ padding: "6px 8px", fontSize: 11 }}>{m.effectiveFrom}</td>
                <td style={{ padding: "6px 8px", fontSize: 11 }}>{m.effectiveTo ?? "-"}</td>
                <td style={{ padding: "6px 8px" }}>
                  <button
                    type="button"
                    onClick={() => void handleRevoke(m.userId, m.areaCode)}
                    style={{ padding: "3px 8px", fontSize: 12, color: "#c00" }}
                  >
                    Revogar
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {!loading && !error && memberships.length === 0 && lastQuery && (
        <div style={{ color: "#888", fontSize: 13 }}>Nenhum membership encontrado.</div>
      )}

      {showGrant && (
        <MembershipGrantDialog
          onClose={() => setShowGrant(false)}
          onGranted={() => {
            setShowGrant(false);
            if (lastQuery) {
              void fetchMemberships(lastQuery).then(setMemberships).catch(() => {});
            }
          }}
        />
      )}
    </div>
  );
}
