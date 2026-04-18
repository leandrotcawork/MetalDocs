import type { Token } from './types';

export interface SchemaDiff {
  used: string[];
  missing: string[];
  orphans: string[];
}

export function diffTokensVsSchema(tokens: Token[], schema: any): SchemaDiff {
  const declared = collectSchemaIdents(schema);
  const referenced = new Set<string>();
  for (const t of tokens) {
    if (t.kind === 'closing') continue;
    referenced.add(t.ident);
  }
  const used: string[] = [];
  const missing: string[] = [];
  for (const d of declared) {
    if (referenced.has(d)) used.push(d); else missing.push(d);
  }
  const orphans: string[] = [];
  for (const r of referenced) {
    if (!declared.has(r)) orphans.push(r);
  }
  return { used, missing, orphans };
}

function collectSchemaIdents(node: any, acc = new Set<string>()): Set<string> {
  if (!node || typeof node !== 'object') return acc;
  if (node.properties && typeof node.properties === 'object') {
    for (const [k, v] of Object.entries(node.properties)) {
      acc.add(k);
      collectSchemaIdents(v, acc);
    }
  }
  if (node.items) collectSchemaIdents(node.items, acc);
  return acc;
}
