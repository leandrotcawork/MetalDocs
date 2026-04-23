import { useEffect, useState } from "react";
import { fetchAreas } from "../taxonomy/api";
import type { ProcessArea } from "../taxonomy/types";
import { grantMembership } from "./membershipApi";
import type { GrantMembershipRequest } from "./membershipApi";

type Props = {
  onClose: () => void;
  onGranted: () => void;
};

const ROLES: GrantMembershipRequest["role"][] = ["viewer", "editor", "reviewer", "approver"];

export function MembershipGrantDialog({ onClose, onGranted }: Props) {
  const [areas, setAreas] = useState<ProcessArea[]>([]);
  const [userId, setUserId] = useState("");
  const [areaCode, setAreaCode] = useState("");
  const [role, setRole] = useState<GrantMembershipRequest["role"]>("viewer");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    void (async () => {
      try {
        const a = await fetchAreas();
        setAreas(a);
      } catch {
        setError("Failed to load areas.");
      }
    })();
  }, []);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSaving(true);
    try {
      await grantMembership({ userId: userId.trim(), areaCode, role });
      onGranted();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to grant.");
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
      <div style={{ background: "#fff", borderRadius: 8, padding: 24, minWidth: 380, maxWidth: 480, width: "100%" }}>
        <h2 style={{ margin: "0 0 16px", fontSize: 16 }}>Conceder acesso de area</h2>
        <form onSubmit={(e) => void handleSubmit(e)}>
          <div style={{ marginBottom: 12 }}>
            <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>User ID *</label>
            <input
              value={userId}
              onChange={(e) => setUserId(e.target.value)}
              required
              style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box" }}
            />
          </div>
          <div style={{ marginBottom: 12 }}>
            <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>Area *</label>
            <select
              value={areaCode}
              onChange={(e) => setAreaCode(e.target.value)}
              required
              style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box" }}
            >
              <option value="">-- Selecionar --</option>
              {areas.map((a) => (
                <option key={a.code} value={a.code}>{a.code} — {a.name}</option>
              ))}
            </select>
          </div>
          <div style={{ marginBottom: 16 }}>
            <label style={{ display: "block", fontSize: 12, marginBottom: 4 }}>Papel *</label>
            <select
              value={role}
              onChange={(e) => setRole(e.target.value as GrantMembershipRequest["role"])}
              style={{ width: "100%", padding: "6px 8px", boxSizing: "border-box" }}
            >
              {ROLES.map((r) => <option key={r} value={r}>{r}</option>)}
            </select>
          </div>
          {error && <p style={{ color: "#c00", fontSize: 12, marginBottom: 8 }}>{error}</p>}
          <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
            <button type="button" onClick={onClose} style={{ padding: "6px 14px" }}>Cancelar</button>
            <button type="submit" disabled={saving} style={{ padding: "6px 14px" }}>
              {saving ? "Concedendo..." : "Conceder"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
