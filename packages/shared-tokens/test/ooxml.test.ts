import { describe, it, expect } from 'vitest';
import { WHITELIST, BLACKLIST, isElementAllowed, classifyBlacklist } from '../src/ooxml';

describe('OOXML lists', () => {
  it('whitelist contains core body elements', () => {
    for (const el of ['w:p','w:r','w:t','w:tab','w:br','w:tbl','w:tr','w:tc','w:pPr','w:rPr','w:hyperlink','w:drawing','w:hdr','w:ftr','w:sectPr']) {
      expect(WHITELIST.has(el)).toBe(true);
    }
  });

  it('blacklist contains tracked changes + SDT + comments', () => {
    for (const el of ['w:ins','w:del','w:moveFrom','w:moveTo','w:sdt','w:sdtContent','w:fldChar','w:altChunk']) {
      expect(BLACKLIST.has(el)).toBe(true);
    }
  });

  it('isElementAllowed rejects blacklisted', () => {
    expect(isElementAllowed('w:ins')).toBe(false);
    expect(isElementAllowed('w:p')).toBe(true);
  });

  it('classifyBlacklist returns stable category', () => {
    expect(classifyBlacklist('w:ins')).toBe('tracked-changes');
    expect(classifyBlacklist('w:sdt')).toBe('structured-document-tag');
    expect(classifyBlacklist('w:altChunk')).toBe('alt-chunk');
  });
});
