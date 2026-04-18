export type TokenKind = 'var' | 'section' | 'inverted' | 'closing';

export interface Token {
  kind: TokenKind;
  ident: string;
  start: number;
  end: number;
  run_id: string;
}

export type ParseError =
  | { type: 'split_across_runs'; run_ids: string[]; token_text: string; auto_fixable: true }
  | { type: 'unsupported_construct'; element: string; location: string; auto_fixable: false }
  | { type: 'reserved_ident'; ident: string; location: string }
  | { type: 'malformed_token'; raw: string; location: string }
  | { type: 'nested_section_too_deep'; ident: string; depth: number }
  | { type: 'unmatched_closing'; ident: string; location: string };

export interface ParseResult {
  tokens: Token[];
  errors: ParseError[];
}
