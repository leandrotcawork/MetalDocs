import { describe, it, expect } from 'vitest';
import { parseDocxTokens } from '../src/parser';
import { makeDocx } from './fixtures';

const TRACKED = `<?xml version="1.0"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:ins w:id="1" w:author="a" w:date="2026-04-18T00:00:00Z">
        <w:r><w:t>inserted</w:t></w:r>
      </w:ins>
    </w:p>
  </w:body>
</w:document>`;

const SDT = `<?xml version="1.0"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:sdt><w:sdtContent><w:p><w:r><w:t>{name}</w:t></w:r></w:p></w:sdtContent></w:sdt>
  </w:body>
</w:document>`;

describe('parseDocxTokens (unsupported OOXML)', () => {
  it('rejects tracked changes', async () => {
    const r = await parseDocxTokens(await makeDocx(TRACKED));
    expect(r.errors.some(e => e.type === 'unsupported_construct' && e.element === 'w:ins')).toBe(true);
  });

  it('rejects SDT even if contains valid tokens', async () => {
    const r = await parseDocxTokens(await makeDocx(SDT));
    expect(r.errors.some(e => e.type === 'unsupported_construct' && e.element === 'w:sdt')).toBe(true);
  });
});
