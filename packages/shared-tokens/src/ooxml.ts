export const WHITELIST: ReadonlySet<string> = new Set([
  'w:p','w:r','w:t','w:tab','w:br',
  'w:tbl','w:tr','w:tc',
  'w:pPr','w:rPr','w:hyperlink','w:drawing',
  'w:hdr','w:ftr','w:sectPr',
]);

export const BLACKLIST: ReadonlySet<string> = new Set([
  'w:ins','w:del','w:moveFrom','w:moveTo',
  'w:sdt','w:sdtContent',
  'w:comment','w:commentReference','w:commentRangeStart','w:commentRangeEnd',
  'w:bookmarkStart','w:bookmarkEnd',
  'w:bidi','w:rtl',
  'w:proofErr','w:smartTag',
  'w:fldSimple','w:fldChar',
  'w:object','w:pict','w:altChunk',
]);

export type BlacklistCategory =
  | 'tracked-changes'
  | 'structured-document-tag'
  | 'comments'
  | 'bookmarks'
  | 'bidi'
  | 'proof-err'
  | 'smart-tag'
  | 'legacy-field'
  | 'legacy-object'
  | 'alt-chunk'
  | 'nested-table'
  | 'unknown';

export function classifyBlacklist(el: string): BlacklistCategory {
  if (el === 'w:ins' || el === 'w:del' || el === 'w:moveFrom' || el === 'w:moveTo') return 'tracked-changes';
  if (el === 'w:sdt' || el === 'w:sdtContent') return 'structured-document-tag';
  if (el.startsWith('w:comment')) return 'comments';
  if (el.startsWith('w:bookmark')) return 'bookmarks';
  if (el === 'w:bidi' || el === 'w:rtl') return 'bidi';
  if (el === 'w:proofErr') return 'proof-err';
  if (el === 'w:smartTag') return 'smart-tag';
  if (el === 'w:fldSimple' || el === 'w:fldChar') return 'legacy-field';
  if (el === 'w:object' || el === 'w:pict') return 'legacy-object';
  if (el === 'w:altChunk') return 'alt-chunk';
  return 'unknown';
}

export function isElementAllowed(el: string): boolean {
  if (BLACKLIST.has(el)) return false;
  return WHITELIST.has(el);
}
