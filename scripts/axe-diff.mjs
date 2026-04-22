#!/usr/bin/env node
/**
 * axe-diff.mjs
 *
 * Compares a fresh axe-core report against the baseline.
 * Reports: added violations, removed violations, moved violations.
 * Exits non-zero if any net-new violations found (critical or serious/moderate unless baselined).
 *
 * Usage:
 *   node scripts/axe-diff.mjs \
 *     --report  test-results/axe-report.json \
 *     --baseline e2e/axe-baseline.json \
 *     [--html test-results/axe-diff.html]
 */

import { readFileSync, writeFileSync } from 'fs';
import { parseArgs } from 'util';

const { values } = parseArgs({
  options: {
    report:   { type: 'string' },
    baseline: { type: 'string' },
    html:     { type: 'string', default: '' },
  },
});

if (!values.report || !values.baseline) {
  console.error('Usage: axe-diff.mjs --report <path> --baseline <path> [--html <path>]');
  process.exit(2);
}

/** @param {string} path */
function load(path) {
  try {
    return JSON.parse(readFileSync(path, 'utf8'));
  } catch (e) {
    console.error(`Failed to load ${path}: ${e.message}`);
    process.exit(2);
  }
}

const report   = load(values.report);    // Array of axe violations
const baseline = load(values.baseline);  // Array of baseline entries

/**
 * Canonical key for a violation: rule id + first node selector.
 * Good enough to detect added/removed without being brittle to node ordering.
 */
function violationKey(v) {
  const firstNode = v.nodes?.[0]?.target?.join(',') ?? v.nodes?.[0] ?? '';
  return `${v.id}::${firstNode}`;
}

const baselineKeys = new Map(baseline.map(b => [violationKey(b), b]));
const reportKeys   = new Map(report.map(v => [violationKey(v), v]));

const added   = [];
const removed = [];
const stayed  = [];

for (const [key, v] of reportKeys) {
  if (baselineKeys.has(key)) {
    stayed.push({ key, v, baseline: baselineKeys.get(key) });
  } else {
    added.push({ key, v });
  }
}

for (const [key, b] of baselineKeys) {
  if (!reportKeys.has(key)) {
    removed.push({ key, baseline: b });
  }
}

// ── Output ──────────────────────────────────────────────────────────────────

let hasBlockingViolation = false;

console.log('\n── axe-diff results ──────────────────────────────────────────');

if (added.length === 0) {
  console.log('✅  No new axe violations added.');
} else {
  console.log(`\n❌  ${added.length} NEW violation(s) detected:\n`);
  for (const { key, v } of added) {
    const impact = v.impact ?? 'unknown';
    const blocking = impact === 'critical' || impact === 'serious' || impact === 'moderate';
    if (blocking) hasBlockingViolation = true;
    const marker = blocking ? '🚫' : '⚠️ ';
    console.log(`  ${marker} [${impact}] ${v.id}`);
    console.log(`       Rule:    ${v.help}`);
    console.log(`       HelpURL: ${v.helpUrl}`);
    const nodes = (v.nodes ?? []).slice(0, 3);
    for (const n of nodes) {
      console.log(`       Node:    ${n.target ?? n}`);
    }
    console.log();
  }
}

if (removed.length > 0) {
  console.log(`\n✨  ${removed.length} violation(s) resolved (can trim baseline):\n`);
  for (const { key, baseline: b } of removed) {
    console.log(`  ✔ ${b.id} (was: ${b.impact})`);
  }
}

if (stayed.length > 0) {
  console.log(`\nℹ️   ${stayed.length} baselined violation(s) unchanged.`);
}

console.log('──────────────────────────────────────────────────────────────\n');

// ── Optional HTML report ───────────────────────────────────────────────────

if (values.html) {
  const rows = (items, color) =>
    items.map(({ key, v, baseline: b }) => `
      <tr style="color:${color}">
        <td>${(v ?? b).id}</td>
        <td>${(v ?? b).impact ?? ''}</td>
        <td>${(v ?? b).help ?? (b?.reason ?? '')}</td>
        <td style="font-size:0.8em">${key}</td>
      </tr>`).join('');

  const html = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>axe-diff report</title>
<style>body{font-family:sans-serif;padding:1rem}table{border-collapse:collapse;width:100%}
th,td{border:1px solid #ccc;padding:0.4rem 0.6rem;text-align:left}
th{background:#f5f5f5}</style></head>
<body>
<h1>axe-diff Report</h1>
<h2 style="color:${hasBlockingViolation ? 'red' : 'green'}">
  ${hasBlockingViolation ? '❌ BLOCKING violations found' : '✅ No blocking violations'}
</h2>
${added.length ? `<h3>Added (${added.length})</h3>
<table><tr><th>Rule</th><th>Impact</th><th>Help</th><th>Key</th></tr>
${rows(added.map(a => ({...a, baseline: undefined})), 'red')}</table>` : ''}
${removed.length ? `<h3>Resolved (${removed.length})</h3>
<table><tr><th>Rule</th><th>Impact</th><th>Reason</th><th>Key</th></tr>
${rows(removed.map(r => ({key: r.key, v: undefined, baseline: r.baseline})), 'green')}</table>` : ''}
${stayed.length ? `<h3>Unchanged baseline (${stayed.length})</h3>
<table><tr><th>Rule</th><th>Impact</th><th>Reason</th><th>Key</th></tr>
${rows(stayed.map(s => ({key: s.key, v: undefined, baseline: s.baseline})), '#555')}</table>` : ''}
</body></html>`;

  writeFileSync(values.html, html);
  console.log(`HTML report written to ${values.html}`);
}

if (hasBlockingViolation) {
  console.error('axe-diff: blocking violations found — failing build.');
  process.exit(1);
}
