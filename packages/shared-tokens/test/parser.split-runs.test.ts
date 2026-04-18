import { describe, it, expect } from 'vitest';
import { parseDocxTokens } from '../src/parser';
import { makeDocx } from './fixtures';

const SPLIT = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r><w:t xml:space="preserve">{clie</w:t></w:r>
      <w:r><w:t xml:space="preserve">nt_na</w:t></w:r>
      <w:r><w:t>me}</w:t></w:r>
    </w:p>
  </w:body>
</w:document>`;

describe('parseDocxTokens (split across runs)', () => {
  it('emits split_across_runs error, no token', async () => {
    const buf = await makeDocx(SPLIT);
    const r = await parseDocxTokens(buf);
    expect(r.tokens).toHaveLength(0);
    expect(r.errors).toHaveLength(1);
    expect(r.errors[0]).toMatchObject({
      type: 'split_across_runs',
      auto_fixable: true,
      token_text: '{client_name}',
    });
    expect((r.errors[0] as any).run_ids.length).toBeGreaterThan(1);
  });
});
