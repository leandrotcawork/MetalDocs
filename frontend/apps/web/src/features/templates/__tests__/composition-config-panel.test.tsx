import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { CompositionConfigPanel } from '../composition-config-panel';
import type { CompositionConfig, SubBlockDef } from '../placeholder-types';

const catalogue: SubBlockDef[] = [
  { key: 'doc_header', label: 'Document Header', params: [{ name: 'show_logo', type: 'boolean' }] },
  { key: 'page_numbers', label: 'Page Numbers', params: [] },
];

const empty: CompositionConfig = {
  headerSubBlocks: [],
  footerSubBlocks: [],
  subBlockParams: {},
};

describe('CompositionConfigPanel', () => {
  it('renders header and footer checkboxes for each catalogue block', () => {
    render(<CompositionConfigPanel value={empty} subBlockCatalogue={catalogue} onChange={vi.fn()} />);
    expect(screen.getByTestId('header-block-doc_header')).toBeInTheDocument();
    expect(screen.getByTestId('footer-block-doc_header')).toBeInTheDocument();
    expect(screen.getByTestId('header-block-page_numbers')).toBeInTheDocument();
  });

  it('toggling header block adds key to headerSubBlocks', () => {
    const onChange = vi.fn();
    render(<CompositionConfigPanel value={empty} subBlockCatalogue={catalogue} onChange={onChange} />);
    fireEvent.click(screen.getByTestId('header-block-doc_header'));
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ headerSubBlocks: ['doc_header'] }),
    );
  });

  it('toggling off removes key from headerSubBlocks', () => {
    const onChange = vi.fn();
    const withHeader: CompositionConfig = { ...empty, headerSubBlocks: ['doc_header'] };
    render(<CompositionConfigPanel value={withHeader} subBlockCatalogue={catalogue} onChange={onChange} />);
    fireEvent.click(screen.getByTestId('header-block-doc_header'));
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ headerSubBlocks: [] }),
    );
  });

  it('shows param inputs when block is enabled', () => {
    const withHeader: CompositionConfig = { ...empty, headerSubBlocks: ['doc_header'] };
    render(<CompositionConfigPanel value={withHeader} subBlockCatalogue={catalogue} onChange={vi.fn()} />);
    expect(screen.getByTestId('param-doc_header-show_logo')).toBeInTheDocument();
  });

  it('editing a param writes into subBlockParams', () => {
    const onChange = vi.fn();
    const withHeader: CompositionConfig = { ...empty, headerSubBlocks: ['doc_header'] };
    render(<CompositionConfigPanel value={withHeader} subBlockCatalogue={catalogue} onChange={onChange} />);
    fireEvent.change(screen.getByTestId('param-doc_header-show_logo'), { target: { value: 'true' } });
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({
        subBlockParams: { doc_header: { show_logo: 'true' } },
      }),
    );
  });
});
