import type { EditableZone } from './placeholder-types';

interface ZoneChipProps {
  zone: EditableZone;
  onInsert?: (z: EditableZone) => void;
}

export function ZoneChip({ zone, onInsert }: ZoneChipProps) {
  return (
    <div
      draggable
      data-testid={`zone-chip-${zone.id}`}
      onDragStart={(e) => {
        e.dataTransfer.setData('application/x-zone-id', zone.id);
        e.dataTransfer.effectAllowed = 'copy';
      }}
      onClick={() => onInsert?.(zone)}
      style={{ cursor: 'grab', display: 'inline-block' }}
    >
      {zone.label}
    </div>
  );
}

export function useZoneDrop(onInsert: (id: string) => void) {
  return {
    onDragOver: (e: React.DragEvent) => {
      e.preventDefault();
      e.dataTransfer.dropEffect = 'copy';
    },
    onDrop: (e: React.DragEvent) => {
      e.preventDefault();
      const id = e.dataTransfer.getData('application/x-zone-id');
      if (id) onInsert(id);
    },
  };
}
