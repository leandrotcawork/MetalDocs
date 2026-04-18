export const IDENT_RE = /^[A-Za-z_][A-Za-z0-9_]*$/;

export const RESERVED_IDENTS = new Set<string>([
  '__proto__',
  'constructor',
  'prototype',
  'toString',
  'valueOf',
  'hasOwnProperty',
  'tenant_id',
  'document_id',
  'template_version_id',
  'revision_id',
  'session_id',
]);

export function isValidIdent(s: string): boolean {
  if (!IDENT_RE.test(s)) return false;
  return true;
}

export function isReservedIdent(s: string): boolean {
  return RESERVED_IDENTS.has(s);
}

export const MAX_SECTION_DEPTH = 1;
