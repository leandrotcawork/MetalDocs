import { describe, it, expect } from 'vitest';
import { injectSentinels } from '../sentinel';

describe('injectSentinels', () => {
  it("injects sentinel as first child for each bid'd block", () => {
    const out = injectSentinels('<p data-mddm-bid="aaa">x</p><p>y</p>');
    expect(out).toContain('<span data-pb-marker="aaa"');
    expect(out.match(/data-pb-marker/g)).toHaveLength(1);
  });
  it('idempotent', () => {
    const once = injectSentinels('<p data-mddm-bid="aaa">x</p>');
    const twice = injectSentinels(once);
    expect(twice.match(/data-pb-marker="aaa"/g)).toHaveLength(1);
  });
});