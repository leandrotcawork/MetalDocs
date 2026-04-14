import type { PublishErrorDTO } from "../../api/templates";

interface ValidationPanelProps {
  errors: PublishErrorDTO[];
  onSelectBlock: (blockId: string) => void;
  onDismiss: () => void;
}

export function ValidationPanel({ errors, onSelectBlock, onDismiss }: ValidationPanelProps) {
  if (errors.length === 0) return null;

  return (
    <div
      data-testid="validation-panel"
      style={{
        position: 'absolute',
        bottom: 0,
        left: 0,
        right: 0,
        zIndex: 100,
        background: 'var(--color-surface-1, #16181f)',
        borderTop: '1px solid rgba(239,68,68,0.4)',
        boxShadow: '0 -4px 24px rgba(0,0,0,0.4)',
        maxHeight: '260px',
        display: 'flex',
        flexDirection: 'column',
        animation: 'slideUpPanel 0.18s ease-out',
      }}
    >
      <style>{`
        @keyframes slideUpPanel {
          from { transform: translateY(100%); opacity: 0; }
          to   { transform: translateY(0);    opacity: 1; }
        }
      `}</style>

      {/* Panel header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '10px 16px',
          borderBottom: '1px solid rgba(239,68,68,0.2)',
          flexShrink: 0,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ color: '#f87171', fontSize: '14px' }}>⚠</span>
          <span style={{ fontSize: '13px', fontWeight: 600, color: '#fca5a5' }}>
            Problemas a corrigir antes de publicar
          </span>
          <span
            style={{
              fontSize: '11px',
              background: 'rgba(239,68,68,0.2)',
              color: '#fca5a5',
              borderRadius: '10px',
              padding: '1px 7px',
              fontWeight: 600,
            }}
          >
            {errors.length}
          </span>
        </div>
        <button
          data-testid="validation-panel-dismiss"
          onClick={onDismiss}
          style={{
            background: 'none',
            border: '1px solid rgba(255,255,255,0.15)',
            borderRadius: '4px',
            color: 'rgba(255,255,255,0.5)',
            cursor: 'pointer',
            fontSize: '12px',
            padding: '3px 10px',
          }}
        >
          Fechar
        </button>
      </div>

      {/* Error rows */}
      <div style={{ overflowY: 'auto', flex: 1 }}>
        {errors.map((err, i) => (
          <button
            key={`${err.blockId}-${i}`}
            data-testid={`validation-error-row-${i}`}
            onClick={() => onSelectBlock(err.blockId)}
            style={{
              display: 'flex',
              alignItems: 'flex-start',
              gap: '10px',
              width: '100%',
              padding: '8px 16px',
              background: 'transparent',
              border: 'none',
              borderBottom: '1px solid rgba(255,255,255,0.05)',
              cursor: 'pointer',
              textAlign: 'left',
              transition: 'background 0.1s',
            }}
            onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.background = 'rgba(239,68,68,0.07)'; }}
            onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = 'transparent'; }}
          >
            <span style={{ fontSize: '11px', color: '#f87171', marginTop: '1px' }}>●</span>
            <span style={{ fontSize: '12px', color: 'rgba(255,255,255,0.7)' }}>
              <span style={{ fontWeight: 600, color: '#fca5a5' }}>{err.blockType}</span>
              {' — '}
              <span style={{ fontStyle: 'italic' }}>{err.field}</span>
              {': '}
              {err.reason}
            </span>
          </button>
        ))}
      </div>
    </div>
  );
}
