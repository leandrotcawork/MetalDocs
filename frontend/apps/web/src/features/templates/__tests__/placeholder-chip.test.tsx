import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { PlaceholderChip } from '../placeholder-chip';
import type { Placeholder } from '../placeholder-types';

const ph: Placeholder = { id: 'p1', label: 'Title', type: 'text' };

describe('PlaceholderChip', () => {
  it('renders with correct testid', () => {
    render(<PlaceholderChip placeholder={ph} />);
    expect(screen.getByTestId('placeholder-chip-p1')).toBeInTheDocument();
  });

  it('dragstart sets placeholder id on dataTransfer', () => {
    render(<PlaceholderChip placeholder={ph} />);
    const chip = screen.getByTestId('placeholder-chip-p1');
    const setData = vi.fn();
    fireEvent.dragStart(chip, { dataTransfer: { setData, effectAllowed: '' } });
    expect(setData).toHaveBeenCalledWith('application/x-placeholder-id', 'p1');
  });

  it('click fires onInsert with placeholder', () => {
    const onInsert = vi.fn();
    render(<PlaceholderChip placeholder={ph} onInsert={onInsert} />);
    fireEvent.click(screen.getByTestId('placeholder-chip-p1'));
    expect(onInsert).toHaveBeenCalledWith(ph);
  });
});
