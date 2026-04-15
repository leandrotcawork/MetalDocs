import { describe, it, expect } from 'vitest';
import { AUTHOR_TOOLBAR, FILL_TOOLBAR } from '../toolbars';

describe('toolbars', () => {
  it('AUTHOR_TOOLBAR includes primitive-insertion buttons', () => {
    expect(AUTHOR_TOOLBAR).toContain('insertMddmSection');
    expect(AUTHOR_TOOLBAR).toContain('insertMddmRepeatable');
    expect(AUTHOR_TOOLBAR).toContain('insertMddmField');
    expect(AUTHOR_TOOLBAR).toContain('insertMddmRichBlock');
    expect(AUTHOR_TOOLBAR).toContain('insertTable');
  });

  it('AUTHOR_TOOLBAR includes exception tools', () => {
    expect(AUTHOR_TOOLBAR).toContain('restrictedEditingException');
    expect(AUTHOR_TOOLBAR).toContain('restrictedEditingExceptionBlock');
  });

  it('FILL_TOOLBAR does not include primitive-insertion or exception-creation', () => {
    expect(FILL_TOOLBAR).not.toContain('insertMddmSection');
    expect(FILL_TOOLBAR).not.toContain('restrictedEditingException');
    expect(FILL_TOOLBAR).not.toContain('restrictedEditingExceptionBlock');
  });

  it('FILL_TOOLBAR includes exception navigation', () => {
    expect(FILL_TOOLBAR).toContain('goToNextRestrictedEditingException');
    expect(FILL_TOOLBAR).toContain('goToPreviousRestrictedEditingException');
  });
});
