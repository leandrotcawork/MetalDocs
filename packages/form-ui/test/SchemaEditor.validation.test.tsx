import { describe, it, expect, vi } from 'vitest';
import { validateJsonSchema } from '../src/SchemaEditor';

vi.mock('@monaco-editor/react', () => ({ default: () => null, Editor: () => null }));

describe('validateJsonSchema', () => {
  it('accepts a valid JSON Schema draft-07', () => {
    const r = validateJsonSchema(JSON.stringify({ type: 'object', properties: { x: { type: 'string' } } }));
    expect(r.valid).toBe(true);
    expect(r.errors).toEqual([]);
  });

  it('reports invalid JSON', () => {
    const r = validateJsonSchema('{bad');
    expect(r.valid).toBe(false);
    expect(r.errors[0]).toMatch(/JSON/);
  });

  it('reports invalid schema (type:object with non-object properties)', () => {
    const r = validateJsonSchema(JSON.stringify({ type: 'object', properties: 'not-an-object' }));
    expect(r.valid).toBe(false);
  });
});
