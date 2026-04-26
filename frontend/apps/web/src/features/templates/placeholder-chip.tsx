import type { Placeholder } from './placeholder-types';

interface PlaceholderChipProps {
  placeholder: Placeholder;
  onInsert?: (p: Placeholder) => void;
}

export function PlaceholderChip({ placeholder, onInsert }: PlaceholderChipProps) {
  return (
    <div
      draggable
      data-testid={`placeholder-chip-${placeholder.id}`}
      onDragStart={(e) => {
        e.dataTransfer.setData('application/x-placeholder-id', placeholder.id);
        e.dataTransfer.setData('application/x-placeholder-name', placeholder.name ?? '');
        e.dataTransfer.effectAllowed = 'copy';
      }}
      onClick={() => onInsert?.(placeholder)}
      style={{ cursor: 'grab', display: 'inline-block' }}
    >
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
