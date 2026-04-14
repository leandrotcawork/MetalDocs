import { describe, test, expect } from "vitest";
import { canInsertBlock } from '../block-palette-rules';

describe('canInsertBlock', () => {
  test('section can insert at root level', () => {
    expect(canInsertBlock('section', null)).toBeNull();
  });
  test('section cannot insert inside section', () => {
    expect(canInsertBlock('section', 'section')).not.toBeNull();
  });
  test('field must be inside section', () => {
    expect(canInsertBlock('field', null)).not.toBeNull();
    expect(canInsertBlock('field', 'section')).toBeNull();
  });
  test('unknown block type is rejected', () => {
    expect(canInsertBlock('UNKNOWN', null)).not.toBeNull();
  });
});
