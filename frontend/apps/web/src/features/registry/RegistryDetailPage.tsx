import { useEffect, useState } from "react";
import { fetchControlledDocument, obsoleteControlledDocument } from "./api";
import type { ControlledDocument } from "./types";

type Props = {
  id: string;
  onBack: () => void;
};

function StatusBadge({ status }: { status: ControlledDocument["status"] }) {
  const colors: Record<ControlledDocument["status"], string> = {
    active: "#2a7a2a",
    obsolete: "#888",
    superseded: "#b87a00",
  };
  return (
    <span style={{ color: colors[status], fontWeight: 600, fontSize: 12 }}>
      {status.toUpperCase()}
    </span>
  );
}

export function RegistryDetailPage({ id, onBack }: Props) {
  const [doc, setDoc] = useState<ControlledDocument | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [actioning, setActioning] = useState(false);

  async function load() {
    setLoading(true);
    try {
      const d = await fetchControlledDocument(id);
      setDoc(d);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load.");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [id]);

  async function handleObsolete() {
    if (!doc || !window.confirm(`Tornar obsoleto o documento "${doc.code}"?`)) return;
    setActioning(true);
    try {
      await obsoleteControlledDocument(id);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to obsolete.");
    } finally {
      setActioning(false);
    }
  }

  if (loading) return <div style={{ padding: 24 }}>Carregando...</div>;
  if (error) return <div style={{ padding: 24, color: "#c00" }}>{error}</div>;
  if (!doc) return null;

  return (
    <div style={{ padding: 24 }}>
      <button type="button" onClick={onBack} style={{ marginBottom: 16, padding: "4px 12px" }}>
        &larr; Voltar
      </button>
      <h2 style={{ margin: "0 0 16px", fontSize: 18 }}>{doc.title}</h2>
      <table style={{ borderCollapse: "collapse", fontSize: 13, width: "100%", maxWidth: 600 }}>
        <tbody>
          {[
            ["ID", doc.id],
            ["Codigo", doc.code],
            ["Perfil", doc.profileCode],
            ["Area", doc.processAreaCode],
            ["Departamento", doc.departmentCode ?? "-"],
            ["Numero de sequencia", doc.sequenceNum != null ? String(doc.sequenceNum) : "-"],
            ["Dono", doc.ownerUserId],
            ["Override template", doc.overrideTemplateVersionId ?? "-"],
            ["Criado em", doc.createdAt],
            ["Atualizado em", doc.updatedAt],
          ].map(([label, value]) => (
            <tr key={label} style={{ borderBottom: "1px solid #f0f0f0" }}>
              <td style={{ padding: "6px 8px", fontWeight: 600, width: 180, color: "#555" }}>{label}</td>
              <td style={{ padding: "6px 8px", fontFamily: "monospace", fontSize: 12 }}>{value}</td>
            </tr>
          ))}
          <tr style={{ borderBottom: "1px solid #f0f0f0" }}>
            <td style={{ padding: "6px 8px", fontWeight: 600, color: "#555" }}>Status</td>
            <td style={{ padding: "6px 8px" }}><StatusBadge status={doc.status} /></td>
          </tr>
        </tbody>
      </table>

      {doc.status === "active" && (
        <div style={{ marginTop: 24 }}>
          <button
            type="button"
            onClick={() => void handleObsolete()}
            disabled={actioning}
            style={{ padding: "6px 14px", color: "#c00", borderColor: "#c00" }}
          >
            {actioning ? "Processando..." : "Tornar obsoleto"}
          </button>
        </div>
      )}
    </div>
  );
}
