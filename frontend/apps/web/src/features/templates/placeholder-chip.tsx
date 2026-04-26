import type { Placeholder } from './placeholder-types';

interface PlaceholderChipProps {
  placeholder: Placeholder;
  onInsert?: (p: Placeholder) => void;
  orphan?: boolean;
}

export function PlaceholderChip({ placeholder, onInsert, orphan = false }: PlaceholderChipProps) {
  return (
    <div
      draggable
      data-testid={`placeholder-chip-${placeholder.id}`}
      data-orphan={orphan ? 'true' : 'false'}
      title={orphan ? 'Token not found in document' : undefined}
      onDragStart={(e) => {
        e.dataTransfer.setData('application/x-placeholder-id', placeholder.id);
        e.dataTransfer.setData('application/x-placeholder-name', placeholder.name ?? '');
        e.dataTransfer.effectAllowed = 'copy';
      }}
      onClick={() => onInsert?.(placeholder)}
      style={{ cursor: 'grab', display: 'inline-block', opacity: orphan ? 0.55 : 1 }}
    >
      {orphan && (
        <span aria-hidden="true" title="Token not found in document" style={{ marginRight: 4 }}>
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z" />
            <path d="M12 9v4" />
            <path d="M12 17h.01" />
          </svg>
        </span>
      )}
      {placeholder.label}
    </div>
  );
}

export function usePlaceholderDrop(onInsert: (id: string, name: string) => void) {
  return {
    onDragOver: (e: React.DragEvent) => {
      e.preventDefault();
      e.dataTransfer.dropEffect = 'copy';
    },
    onDrop: (e: React.DragEvent) => {
      e.preventDefault();
      const id   = e.dataTransfer.getData('application/x-placeholder-id');
      const name = e.dataTransfer.getData('application/x-placeholder-name');
      if (id) onInsert(id, name);
    },
  };
}
