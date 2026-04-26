import React, { useEffect, useState } from 'react';
import { fetchPlaceholderCatalog, type PlaceholderCatalogEntry } from './api/catalog';

export interface PlaceholderCatalogPanelProps {
  detected: string[];
}

export function PlaceholderCatalogPanel({ detected }: PlaceholderCatalogPanelProps): React.ReactElement {
  const [items, setItems] = useState<PlaceholderCatalogEntry[]>([]);
  useEffect(() => { void fetchPlaceholderCatalog().then(setItems); }, []);

  const detectedSet = new Set(detected);
  return (
    <aside style={{ width: 280, borderLeft: '1px solid #e2e8f0', padding: 12 }}>
      <h3>Placeholders disponíveis</h3>
      <p style={{ fontSize: 12, color: '#64748b' }}>
        Digite o nome entre chaves no documento, ex.: {'{doc_code}'}
      </p>
      <ul style={{ listStyle: 'none', padding: 0 }}>
        {items.map((it) => (
          <li
            key={it.key}
            data-testid={`catalog-${it.key}`}
            data-detected={detectedSet.has(it.key)}
            style={{
              padding: 6, borderRadius: 4, marginBottom: 4,
              background: detectedSet.has(it.key) ? '#dcfce7' : '#f1f5f9',
            }}
          >
            <code>{`{${it.key}}`}</code>
            <div style={{ fontSize: 11, color: '#475569' }}>{it.label}</div>
          </li>
        ))}
      </ul>
    </aside>
  );
}
