import type { EditableZone } from './placeholder-types';

interface ZoneInspectorProps {
  value: EditableZone;
  onChange: (updated: EditableZone) => void;
}

export function ZoneInspector({ value, onChange }: ZoneInspectorProps) {
  function setPolicy(field: keyof EditableZone['contentPolicy'], checked: boolean) {
    onChange({ ...value, contentPolicy: { ...value.contentPolicy, [field]: checked } });
  }

  return (
    <div data-testid="zone-inspector">
      <label>
        Label
        <input
          data-testid="zone-label"
          type="text"
          value={value.label}
          onChange={(e) => onChange({ ...value, label: e.target.value })}
        />
      </label>

      <label>
        Max length
        <input
          data-testid="zone-maxlength"
          type="number"
          value={value.maxLength ?? ''}
          onChange={(e) =>
            onChange({ ...value, maxLength: e.target.value ? Number(e.target.value) : undefined })
          }
        />
      </label>

      <label>
        Allow tables
        <input
          data-testid="zone-allow-tables"
          type="checkbox"
          checked={value.contentPolicy.allowTables}
          onChange={(e) => setPolicy('allowTables', e.target.checked)}
        />
      </label>

      <label>
        Allow images
        <input
          data-testid="zone-allow-images"
          type="checkbox"
          checked={value.contentPolicy.allowImages}
          onChange={(e) => setPolicy('allowImages', e.target.checked)}
        />
      </label>

      <label>
        Allow headings
        <input
          data-testid="zone-allow-headings"
          type="checkbox"
          checked={value.contentPolicy.allowHeadings}
          onChange={(e) => setPolicy('allowHeadings', e.target.checked)}
        />
      </label>

      <label>
        Allow lists
        <input
          data-testid="zone-allow-lists"
          type="checkbox"
          checked={value.contentPolicy.allowLists}
          onChange={(e) => setPolicy('allowLists', e.target.checked)}
        />
      </label>
    </div>
  );
}
