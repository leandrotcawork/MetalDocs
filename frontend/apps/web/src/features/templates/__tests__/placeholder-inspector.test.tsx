import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { PlaceholderInspector } from '../placeholder-inspector';
import type { Placeholder } from '../placeholder-types';

const textPH: Placeholder = { id: 'p1', label: 'Title', type: 'text' };
const resolvers = [
  { key: 'doc_code', version: 1 },
  { key: 'revision_number', version: 1 },
];

describe('PlaceholderInspector', () => {
  it('renders type=text with label/required/maxLength/regex inputs', () => {
    render(<PlaceholderInspector value={textPH} resolvers={resolvers} onChange={vi.fn()} />);
    expect(screen.getByTestId('ph-label')).toBeInTheDocument();
    expect(screen.getByTestId('ph-type')).toBeInTheDocument();
    expect(screen.getByTestId('ph-required')).toBeInTheDocument();
    expect(screen.getByTestId('ph-maxlength')).toBeInTheDocument();
    expect(screen.getByTestId('ph-regex')).toBeInTheDocument();
    expect(screen.queryByTestId('ph-resolver-key')).not.toBeInTheDocument();
  });

  it('type=number shows minNumber/maxNumber, hides regex', () => {
    const ph: Placeholder = { id: 'p1', label: 'Qty', type: 'number' };
    render(<PlaceholderInspector value={ph} resolvers={resolvers} onChange={vi.fn()} />);
    expect(screen.getByTestId('ph-min-number')).toBeInTheDocument();
    expect(screen.getByTestId('ph-max-number')).toBeInTheDocument();
    expect(screen.queryByTestId('ph-regex')).not.toBeInTheDocument();
  });

  it('type=computed shows resolver-key select, hides constraint fields', () => {
    const ph: Placeholder = { id: 'p1', label: 'Code', type: 'computed' };
    render(<PlaceholderInspector value={ph} resolvers={resolvers} onChange={vi.fn()} />);
    expect(screen.getByTestId('ph-resolver-key')).toBeInTheDocument();
    expect(screen.queryByTestId('ph-maxlength')).not.toBeInTheDocument();
    expect(screen.queryByTestId('ph-regex')).not.toBeInTheDocument();
    expect(screen.queryByTestId('ph-required')).not.toBeInTheDocument();
  });

  it('resolver-key select is populated from resolvers prop', () => {
    const ph: Placeholder = { id: 'p1', label: 'Code', type: 'computed' };
    render(<PlaceholderInspector value={ph} resolvers={resolvers} onChange={vi.fn()} />);
    expect(screen.getByRole('option', { name: /doc_code/ })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: /revision_number/ })).toBeInTheDocument();
  });

  it('onChange fires with merged object when label changes', () => {
    const onChange = vi.fn();
    render(<PlaceholderInspector value={textPH} resolvers={resolvers} onChange={onChange} />);
    fireEvent.change(screen.getByTestId('ph-label'), { target: { value: 'New Label' } });
    expect(onChange).toHaveBeenCalledWith({ ...textPH, label: 'New Label', name: 'new_label' });
  });

  it('type=select shows options textarea', () => {
    const ph: Placeholder = { id: 'p1', label: 'Status', type: 'select', options: ['A', 'B'] };
    render(<PlaceholderInspector value={ph} resolvers={resolvers} onChange={vi.fn()} />);
    expect(screen.getByTestId('ph-options')).toBeInTheDocument();
    expect((screen.getByTestId('ph-options') as HTMLTextAreaElement).value).toBe('A\nB');
  });

  it('type=date shows min/max date, hides regex', () => {
    const ph: Placeholder = { id: 'p1', label: 'Date', type: 'date' };
    render(<PlaceholderInspector value={ph} resolvers={resolvers} onChange={vi.fn()} />);
    expect(screen.getByTestId('ph-min-date')).toBeInTheDocument();
    expect(screen.getByTestId('ph-max-date')).toBeInTheDocument();
    expect(screen.queryByTestId('ph-regex')).not.toBeInTheDocument();
  });
});
