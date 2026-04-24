import type { CompositionConfig, SubBlockDef } from './placeholder-types';

interface CompositionConfigPanelProps {
  value: CompositionConfig;
  subBlockCatalogue: SubBlockDef[];
  onChange: (updated: CompositionConfig) => void;
}

export function CompositionConfigPanel({ value, subBlockCatalogue, onChange }: CompositionConfigPanelProps) {
  function toggleBlock(section: 'headerSubBlocks' | 'footerSubBlocks', key: string, enabled: boolean) {
    const current = value[section];
    const next = enabled ? [...current, key] : current.filter((k) => k !== key);
    onChange({ ...value, [section]: next });
  }

  function setParam(blockKey: string, paramName: string, paramValue: string) {
    const existing = value.subBlockParams[blockKey] ?? {};
    onChange({
      ...value,
      subBlockParams: {
        ...value.subBlockParams,
        [blockKey]: { ...existing, [paramName]: paramValue },
      },
    });
  }

  return (
    <div data-testid="composition-config">
      <section>
        <h4>Header sub-blocks</h4>
        {subBlockCatalogue.map((block) => {
          const enabled = value.headerSubBlocks.includes(block.key);
          return (
            <div key={block.key}>
              <label>
                <input
                  data-testid={`header-block-${block.key}`}
                  type="checkbox"
                  checked={enabled}
                  onChange={(e) => toggleBlock('headerSubBlocks', block.key, e.target.checked)}
                />
                {block.label}
              </label>
              {enabled && block.params.map((param) => (
                <label key={param.name}>
                  {param.name}
                  <input
                    data-testid={`param-${block.key}-${param.name}`}
                    type={param.type === 'number' ? 'number' : 'text'}
                    value={value.subBlockParams[block.key]?.[param.name] ?? ''}
                    onChange={(e) => setParam(block.key, param.name, e.target.value)}
                  />
                </label>
              ))}
            </div>
          );
        })}
      </section>

      <section>
        <h4>Footer sub-blocks</h4>
        {subBlockCatalogue.map((block) => {
          const enabled = value.footerSubBlocks.includes(block.key);
          return (
            <div key={block.key}>
              <label>
                <input
                  data-testid={`footer-block-${block.key}`}
                  type="checkbox"
                  checked={enabled}
                  onChange={(e) => toggleBlock('footerSubBlocks', block.key, e.target.checked)}
                />
                {block.label}
              </label>
              {enabled && block.params.map((param) => (
                <label key={param.name}>
                  {param.name}
                  <input
                    data-testid={`param-${block.key}-${param.name}`}
                    type={param.type === 'number' ? 'number' : 'text'}
                    value={value.subBlockParams[block.key]?.[param.name] ?? ''}
                    onChange={(e) => setParam(block.key, param.name, e.target.value)}
                  />
                </label>
              ))}
            </div>
          );
        })}
      </section>
    </div>
  );
}
