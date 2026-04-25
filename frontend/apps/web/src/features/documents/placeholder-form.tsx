import { useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import type { Placeholder } from '../templates/placeholder-types';
import type { PlaceholderValueDTO } from './v2/api/documentsV2';

interface PlaceholderFormProps {
  schema: Placeholder[];
  values: PlaceholderValueDTO[];
  disabled?: boolean;
  onSave: (placeholderId: string, value: string) => Promise<void>;
}

function buildInitial(schema: Placeholder[], values: PlaceholderValueDTO[]): Record<string, string> {
  const init: Record<string, string> = {};
  for (const p of schema) {
    init[p.id] = values.find((v) => v.placeholder_id === p.id)?.value_text ?? '';
  }
  return init;
}

export function PlaceholderForm({ schema, values, disabled, onSave }: PlaceholderFormProps) {
  const [localValues, setLocalValues] = useState<Record<string, string>>(() => buildInitial(schema, values));

  useEffect(() => {
    setLocalValues(buildInitial(schema, values));
  }, [schema, values]);

  return (
    <div>
      {schema.map((p) => {
        const fieldId = `ph-${p.id}`;
        const value = localValues[p.id] ?? '';
        const serverValue = values.find((v) => v.placeholder_id === p.id)?.value_text ?? '';

        const handleChange = (next: string) => {
          setLocalValues((prev) => ({ ...prev, [p.id]: next }));
        };

        const handleBlur = () => {
          if (value === serverValue) return;
          void onSave(p.id, value);
        };

        const opts = p.options;
        let input: ReactElement;
        if (p.type === 'select' && opts && opts.length > 0) {
          input = (
            <select
              id={fieldId}
              value={value}
              disabled={disabled}
              onChange={(e) => handleChange(e.target.value)}
              onBlur={handleBlur}
              style={{ width: '100%' }}
            >
              <option value="">—</option>
              {opts.map((opt) => (
                <option key={opt} value={opt}>{opt}</option>
              ))}
            </select>
          );
        } else if (p.type === 'date') {
          input = (
            <input
              id={fieldId}
              type="date"
              value={value}
              disabled={disabled}
              onChange={(e) => handleChange(e.target.value)}
              onBlur={handleBlur}
              style={{ width: '100%' }}
            />
          );
        } else if (p.type === 'number') {
          input = (
            <input
              id={fieldId}
              type="number"
              value={value}
              disabled={disabled}
              onChange={(e) => handleChange(e.target.value)}
              onBlur={handleBlur}
              style={{ width: '100%' }}
            />
          );
        } else {
          input = (
            <input
              id={fieldId}
              type="text"
              value={value}
              disabled={disabled}
              onChange={(e) => handleChange(e.target.value)}
              onBlur={handleBlur}
              style={{ width: '100%' }}
            />
          );
        }

        return (
          <div key={p.id} style={{ marginBottom: 12 }}>
            <label htmlFor={fieldId} style={{ display: 'block', fontSize: 12, marginBottom: 4 }}>
              {p.label}{p.required && <span style={{ color: 'red', marginLeft: 2 }}>*</span>}
            </label>
            {input}
          </div>
        );
      })}
    </div>
  );
}
