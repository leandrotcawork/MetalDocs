import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { ZoneChip } from '../zone-chip';
import type { EditableZone } from '../placeholder-types';

const zone: EditableZone = {
  id: 'z1',
  label: 'Section A',
  contentPolicy: { allowTables: true, allowImages: true, allowHeadings: true, allowLists: true },
};

describe('ZoneChip', () => {
  it('renders with correct testid', () => {
    render(<ZoneChip zone={zone} />);
    expect(screen.getByTestId('zone-chip-z1')).toBeInTheDocument();
  });

  it('dragstart sets zone id on dataTransfer', () => {
    render(<ZoneChip zone={zone} />);
    const chip = screen.getByTestId('zone-chip-z1');
    const setData = vi.fn();
    fireEvent.dragStart(chip, { dataTransfer: { setData, effectAllowed: '' } });
    expect(setData).toHaveBeenCalledWith('application/x-zone-id', 'z1');
  });

  it('click fires onInsert with zone', () => {
    const onInsert = vi.fn();
    render(<ZoneChip zone={zone} onInsert={onInsert} />);
    fireEvent.click(screen.getByTestId('zone-chip-z1'));
    expect(onInsert).toHaveBeenCalledWith(zone);
  });
});
