import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { fireEvent } from '@testing-library/react';
import { LockBadge } from './LockBadge';

describe('LockBadge', () => {
  it('renders nothing when no instance id', () => {
    const { container } = render(<LockBadge />);
    expect(container.firstChild).toBeNull();
  });

  it('shows locked banner with actor', () => {
    render(<LockBadge lockedByInstanceId="inst-1" lockedByActor="Alice" />);
    expect(screen.getByText(/Alice/)).toBeTruthy();
  });

  it('calls onBannerClick when clicked', () => {
    const spy = vi.fn();
    render(<LockBadge lockedByInstanceId="inst-1" lockedByActor="Alice" onBannerClick={spy} />);
    fireEvent.click(screen.getByRole('button'));
    expect(spy).toHaveBeenCalledOnce();
  });
});
