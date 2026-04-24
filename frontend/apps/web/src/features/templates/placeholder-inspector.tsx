import type { Placeholder, PlaceholderType } from './placeholder-types';

interface PlaceholderInspectorProps {
  value: Placeholder;
  resolvers: { key: string; version: number }[];
  onChange: (updated: Placeholder) => void;
}

const ALL_TYPES: PlaceholderType[] = ['text', 'date', 'number', 'select', 'user', 'picture', 'computed'];

export function PlaceholderInspector({ value, resolvers, onChange }: PlaceholderInspectorProps) {
  function set<K extends keyof Placeholder>(field: K, val: Placeholder[K]) {
    onChange({ ...value, [field]: val });
  }

  const isComputed = value.type === 'computed';
  const isNumber = value.type === 'number';
  const isDate = value.type === 'date';
  const isSelect = value.type === 'select';
  const showTextConstraints = !isComputed && !isNumber && !isDate && !isSelect && value.type !== 'user' && value.type !== 'picture';

  return (
    <div data-testid="placeholder-inspector">
      <label>
        Label
        <input
          data-testid="ph-label"
          type="text"
          value={value.label}
          onChange={(e) => set('label', e.target.value)}
        />
      </label>

      <label>
        Type
        <select
          data-testid="ph-type"
          value={value.type}
          onChange={(e) => set('type', e.target.value as PlaceholderType)}
        >
          {ALL_TYPES.map((t) => (
            <option key={t} value={t}>{t}</option>
          ))}
        </select>
      </label>

      {!isComputed && (
        <label>
          Required
          <input
            data-testid="ph-required"
            type="checkbox"
            checked={!!value.required}
            onChange={(e) => set('required', e.target.checked)}
          />
        </label>
      )}

      {(showTextConstraints || isDate) && (
        <>
          {showTextConstraints && (
            <>
              <label>
                Max length
                <input
                  data-testid="ph-maxlength"
                  type="number"
                  value={value.maxLength ?? ''}
                  onChange={(e) => set('maxLength', e.target.value ? Number(e.target.value) : undefined)}
                />
              </label>
              <label>
                Regex
                <input
                  data-testid="ph-regex"
                  type="text"
                  value={value.regex ?? ''}
                  onChange={(e) => set('regex', e.target.value || undefined)}
                />
              </label>
            </>
          )}
          {isDate && (
            <>
              <label>
                Min date
                <input
                  data-testid="ph-min-date"
                  type="text"
                  placeholder="YYYY-MM-DD"
                  value={value.minDate ?? ''}
                  onChange={(e) => set('minDate', e.target.value || undefined)}
                />
              </label>
              <label>
                Max date
                <input
                  data-testid="ph-max-date"
                  type="text"
                  placeholder="YYYY-MM-DD"
                  value={value.maxDate ?? ''}
                  onChange={(e) => set('maxDate', e.target.value || undefined)}
                />
              </label>
            </>
          )}
        </>
      )}

      {isNumber && (
        <>
          <label>
            Min number
            <input
              data-testid="ph-min-number"
              type="number"
              value={value.minNumber ?? ''}
              onChange={(e) => set('minNumber', e.target.value ? Number(e.target.value) : undefined)}
            />
          </label>
          <label>
            Max number
            <input
              data-testid="ph-max-number"
              type="number"
              value={value.maxNumber ?? ''}
              onChange={(e) => set('maxNumber', e.target.value ? Number(e.target.value) : undefined)}
            />
          </label>
        </>
      )}

      {isSelect && (
        <label>
          Options (one per line)
          <textarea
            data-testid="ph-options"
            value={(value.options ?? []).join('\n')}
            onChange={(e) =>
              set('options', e.target.value ? e.target.value.split('\n').filter(Boolean) : [])
            }
          />
        </label>
      )}

      {isComputed && (
        <label>
          Resolver key
          <select
            data-testid="ph-resolver-key"
            value={value.resolverKey ?? ''}
            onChange={(e) => set('resolverKey', e.target.value || undefined)}
          >
            <option value="">— select —</option>
            {resolvers.map((r) => (
              <option key={r.key} value={r.key}>{r.key} v{r.version}</option>
            ))}
          </select>
        </label>
      )}
    </div>
  );
}
