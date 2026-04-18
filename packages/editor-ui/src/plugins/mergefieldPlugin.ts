import { diffTokensVsSchema, classifyBlacklist, type ParseError, type Token } from '@metaldocs/shared-tokens';

export interface SidebarModel {
  used: string[];
  missing: string[];
  orphans: string[];
  bannerError: boolean;
  errorCategories: string[];
}

export function computeSidebarModel(
  tokens: Token[],
  errors: ParseError[],
  schema: unknown
): SidebarModel {
  const diff = diffTokensVsSchema(tokens, schema);
  const bannerError = errors.length > 0;
  const errorCategories = Array.from(new Set(
    errors
      .filter((e): e is Extract<ParseError, { type: 'unsupported_construct' }> => e.type === 'unsupported_construct')
      .map(e => classifyBlacklist(e.element))
  ));
  return {
    used: diff.used,
    missing: diff.missing,
    orphans: diff.orphans,
    bannerError,
    errorCategories,
  };
}
