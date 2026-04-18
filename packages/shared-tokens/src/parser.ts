import JSZip from 'jszip';
import { XMLParser } from 'fast-xml-parser';
import type { ParseError, ParseResult, Token } from './types';
import { MAX_SECTION_DEPTH, isReservedIdent, isValidIdent } from './grammar';
import { BLACKLIST } from './ooxml';

const TOKEN_RE = /\{([#^/])?([^{}]+)\}/g;

interface Run {
  id: string;
  text: string;
  start: number;
  end: number;
}

export async function parseDocxTokens(buf: ArrayBuffer): Promise<ParseResult> {
  const zip = await JSZip.loadAsync(buf);
  const xmlStr = await zip.file('word/document.xml')?.async('string');
  if (!xmlStr) {
    return { tokens: [], errors: [{ type: 'malformed_token', raw: 'word/document.xml missing', location: 'archive' }] };
  }

  const runs: Run[] = [];
  const errors: ParseError[] = [];

  const xp = new XMLParser({ ignoreAttributes: false, preserveOrder: true, trimValues: false });
  const tree = xp.parse(xmlStr) as unknown[];
  walkForRunsAndBadElements(tree, runs, errors, 0);

  const tokens = scanTokens(runs, errors);

  return { tokens, errors };
}

function walkForRunsAndBadElements(node: unknown, runs: Run[], errors: ParseError[], depth: number): void {
  if (!node || typeof node !== 'object') return;
  if (Array.isArray(node)) {
    for (const child of node) walkForRunsAndBadElements(child, runs, errors, depth);
    return;
  }
  const obj = node as Record<string, unknown>;
  for (const [key, value] of Object.entries(obj)) {
    if (key === ':@') continue;
    if (BLACKLIST.has(key)) {
      errors.push({
        type: 'unsupported_construct',
        element: key,
        location: `depth=${depth}`,
        auto_fixable: false,
      });
    }
    if (key === 'w:r') {
      const run: Run = { id: `run_${runs.length}`, text: collectRunText(value), start: 0, end: 0 };
      run.end = run.text.length;
      runs.push(run);
      continue;
    }
    walkForRunsAndBadElements(value, runs, errors, depth + 1);
  }
}

function collectRunText(runNode: unknown): string {
  let out = '';
  const walk = (n: unknown) => {
    if (!n) return;
    if (Array.isArray(n)) { for (const c of n) walk(c); return; }
    if (typeof n === 'object') {
      for (const [k, v] of Object.entries(n as Record<string, unknown>)) {
        if (k === 'w:t') {
          // With preserveOrder: true, w:t value is an array like [{ '#text': 'Hello {client_name}' }]
          if (Array.isArray(v)) {
            for (const it of v) {
              if (typeof it === 'object' && it !== null && '#text' in it) {
                out += String((it as { '#text': unknown })['#text']);
              }
            }
          } else if (typeof v === 'string') {
            out += v;
          }
        } else if (k === '#text') {
          out += String(v);
        } else if (k !== ':@') {
          walk(v);
        }
      }
    }
  };
  walk(runNode);
  return out;
}

function scanTokens(runs: Run[], errors: ParseError[]): Token[] {
  const full = runs.map(r => r.text).join('');
  const positions = runPositions(runs);
  const tokens: Token[] = [];
  const openSections: string[] = [];

  TOKEN_RE.lastIndex = 0;
  let m: RegExpExecArray | null;
  while ((m = TOKEN_RE.exec(full)) !== null) {
    const [raw, prefix, inner] = m;
    const ident = inner.trim();
    const start = m.index;
    const end = start + raw.length;

    const spanningRunIds = runsSpanning(positions, start, end);
    if (spanningRunIds.length > 1) {
      errors.push({ type: 'split_across_runs', run_ids: spanningRunIds, token_text: raw, auto_fixable: true });
      continue;
    }

    if (!isValidIdent(ident)) {
      errors.push({ type: 'malformed_token', raw, location: `offset=${start}` });
      continue;
    }
    if (isReservedIdent(ident)) {
      errors.push({ type: 'reserved_ident', ident, location: `offset=${start}` });
      continue;
    }

    let kind: Token['kind'];
    if (prefix === '#') kind = 'section';
    else if (prefix === '^') kind = 'inverted';
    else if (prefix === '/') kind = 'closing';
    else kind = 'var';

    if (kind === 'section' || kind === 'inverted') {
      openSections.push(ident);
      if (openSections.length > MAX_SECTION_DEPTH + 1) {
        errors.push({ type: 'nested_section_too_deep', ident, depth: openSections.length - 1 });
      }
    } else if (kind === 'closing') {
      const top = openSections.pop();
      if (top !== ident) {
        errors.push({ type: 'unmatched_closing', ident, location: `offset=${start}` });
      }
    }

    tokens.push({ kind, ident, start, end, run_id: spanningRunIds[0] ?? 'run_0' });
  }

  for (const unclosed of openSections) {
    errors.push({ type: 'unmatched_closing', ident: unclosed, location: 'unclosed-section' });
  }

  return tokens;
}

function runPositions(runs: Run[]): Run[] {
  let cursor = 0;
  return runs.map(r => {
    const start = cursor;
    const end = start + r.text.length;
    cursor = end;
    return { ...r, start, end };
  });
}

function runsSpanning(runs: Run[], start: number, end: number): string[] {
  return runs.filter(r => !(r.end <= start || r.start >= end)).map(r => r.id);
}
