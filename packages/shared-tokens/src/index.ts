export { parseDocxTokens } from './parser';
export { diffTokensVsSchema, type SchemaDiff } from './diff';
export { WHITELIST, BLACKLIST, classifyBlacklist, isElementAllowed } from './ooxml';
export { IDENT_RE, RESERVED_IDENTS, MAX_SECTION_DEPTH, isValidIdent, isReservedIdent } from './grammar';
export type { Token, TokenKind, ParseError, ParseResult } from './types';