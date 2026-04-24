import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { ZoneInspector } from '../zone-inspector';
import type { EditableZone } from '../placeholder-types';

const zone: EditableZone = {
  id: 'z1',
  label: 'Section',
  contentPolicy: { allowTables: false, allowImages: true, allowHeadings: true, allowLists: true },
};

describe('ZoneInspector', () => {
  it('renders all policy checkboxes and inputs', () => {
    render(<ZoneInspector value={zone} onChange={vi.fn()} />);
    expect(screen.getByTestId('zone-label')).toBeInTheDocument();
    expect(screen.getByTestId('zone-maxlength')).toBeInTheDocument();
    expect(screen.getByTestId('zone-allow-tables')).toBeInTheDocument();
    expect(screen.getByTestId('zone-allow-images')).toBeInTheDocument();
    expect(screen.getByTestId('zone-allow-headings')).toBeInTheDocument();
    expect(screen.getByTestId('zone-allow-lists')).toBeInTheDocument();
  });

  it('toggling allow-tables fires onChange with updated contentPolicy', () => {
    const onChange = vi.fn();
    render(<ZoneInspector value={zone} onChange={onChange} />);
    fireEvent.click(screen.getByTestId('zone-allow-tables'));
    expect(onChange).toHaveBeenCalledWith({
      ...zone,
      contentPolicy: { ...zone.contentPolicy, allowTables: true },
    });
  });

  it('editing maxLength fires onChange with number', () => {
    const onChange = vi.fn();
    render(<ZoneInspector value={zone} onChange={onChange} />);
    fireEvent.change(screen.getByTestId('zone-maxlength'), { target: { value: '500' } });
    expect(onChange).toHaveBeenCalledWith({ ...zone, maxLength: 500 });
  });

  it('reflects initial checkbox state', () => {
    render(<ZoneInspector value={zone} onChange={vi.fn()} />);
    expect((screen.getByTestId('zone-allow-tables') as HTMLInputElement).checked).toBe(false);
    expect((screen.getByTestId('zone-allow-images') as HTMLInputElement).checked).toBe(true);
  });
});
