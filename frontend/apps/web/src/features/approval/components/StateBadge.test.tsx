import { render, screen } from '@testing-library/react';
import { describe, expect, it, test } from 'vitest';
import type { ApprovalState } from '../api/approvalTypes';
import { StateBadge } from './StateBadge';

const states: ApprovalState[] = [
  'draft',
  'under_review',
  'approved',
  'scheduled',
  'published',
  'superseded',
  'rejected',
  'obsolete',
  'cancelled',
];

describe('StateBadge', () => {
  test.each(states)('renders %s with accessible label', (state: ApprovalState) => {
    render(<StateBadge state={state} />);
    expect(screen.getByRole('generic', { name: /Estado:/i })).toBeTruthy();
  });

  it('applies sm size class', () => {
    const { container } = render(<StateBadge state="draft" size="sm" />);
    expect((container.firstChild as HTMLElement).className.includes('sm')).toBe(true);
  });

  it('renders expected label for approved state', () => {
    render(<StateBadge state="approved" />);
    expect(screen.getByText('Aprovado')).toBeTruthy();
  });
});
