import { useState } from "react";
import { acknowledgeStripped } from "../../api/templates";
import type { TemplateDraftDTO } from "../../api/templates";

interface StrippedFieldsBannerProps {
  templateKey: string;
  lockVersion: number;
  /** Called with the updated draft returned by the server after acknowledgement. */
  onAcknowledged: (updatedDraft: TemplateDraftDTO) => void;
}

export function StrippedFieldsBanner({ templateKey, lockVersion, onAcknowledged }: StrippedFieldsBannerProps) {
  const [isLoading, setIsLoading] = useState(false);
  const [apiError, setApiError] = useState<string | null>(null);

  async function handleAcknowledge() {
    setIsLoading(true);
    setApiError(null);
    try {
      const updatedDraft = await acknowledgeStripped(templateKey, lockVersion);
      onAcknowledged(updatedDraft);
    } catch (err) {
      setApiError(err instanceof Error ? err.message : 'Erro ao reconhecer alterações.');
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <div
      data-testid="stripped-fields-banner"
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '12px',
        padding: '8px 16px',
        background: 'rgba(251,191,36,0.1)',
        borderBottom: '1px solid rgba(251,191,36,0.3)',
        flexShrink: 0,
      }}
    >
      <span style={{ fontSize: '14px', flexShrink: 0 }}>⚠</span>
      <span style={{ fontSize: '12px', color: 'rgba(255,255,255,0.8)', flex: 1 }}>
        Este template foi importado com campos desconhecidos que foram removidos.
        Reconheça as alterações para poder publicar.
      </span>
      {apiError && (
        <span style={{ fontSize: '11px', color: '#f87171', flexShrink: 0 }}>{apiError}</span>
      )}
      <button
        data-testid="stripped-fields-acknowledge-btn"
        onClick={() => void handleAcknowledge()}
        disabled={isLoading}
        style={{
          flexShrink: 0,
          padding: '4px 12px',
          fontSize: '12px',
          fontWeight: 600,
          background: isLoading ? 'rgba(251,191,36,0.15)' : 'rgba(251,191,36,0.2)',
          border: '1px solid rgba(251,191,36,0.5)',
          borderRadius: '4px',
          color: '#fcd34d',
          cursor: isLoading ? 'wait' : 'pointer',
          transition: 'background 0.1s',
        }}
      >
        {isLoading ? 'Processando...' : 'Reconheço as alterações'}
      </button>
    </div>
  );
}
