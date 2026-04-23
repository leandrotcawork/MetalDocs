import { describe, expect, test } from 'vitest';
import { SubBlockRegistry } from '../registry';

describe('SubBlockRegistry', () => {
  test('register and render returns ooxml', async () => {
    const reg = new SubBlockRegistry();
    reg.register({
      key: 'k1',
      render: async () => '<w:p>X</w:p>',
    });
    const out = await reg.render('k1', { params: {}, values: {} });
    expect(out).toBe('<w:p>X</w:p>');
  });

  test('unknown key throws', async () => {
    const reg = new SubBlockRegistry();
    await expect(reg.render('nope', { params: {}, values: {} })).rejects.toThrow(/unknown/);
  });
});
