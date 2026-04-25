import React from 'react';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { SubmitButton } from '../submit-button';
import * as api from '../v2/api/documentsV2';

vi.mock('../v2/api/documentsV2');

const schema = [
  { id: 'p1', label: 'Document Title', type: 'text' as const, required: true },
  { id: 'p2', label: 'Author', type: 'user' as const, required: false },
];

describe('SubmitButton', () => {
  it('disabled with missing required list when required placeholder empty', () => {
    render(
      <SubmitButton
        docId="doc-1"
        placeholderSchema={schema}
        placeholderValues={[]}
      />,
    );
    expect(screen.getByTestId('submit-btn')).toBeDisabled();
    expect(screen.getByTestId('missing-required-list')).toBeInTheDocument();
    expect(screen.getByText('Document Title')).toBeInTheDocument();
  });

  it('enabled when all required placeholders filled', () => {
    render(
      <SubmitButton
        docId="doc-1"
        placeholderSchema={schema}
        placeholderValues={[{ placeholder_id: 'p1', value_text: 'My Doc', source: 'user' }]}
      />,
    );
    expect(screen.getByTestId('submit-btn')).not.toBeDisabled();
    expect(screen.queryByTestId('missing-required-list')).not.toBeInTheDocument();
  });

  it('calls submitDocument on click when all filled', async () => {
    vi.mocked(api.submitDocument).mockResolvedValue(undefined);
    const onSubmitted = vi.fn();

    render(
      <SubmitButton
        docId="doc-1"
        placeholderSchema={schema}
        placeholderValues={[{ placeholder_id: 'p1', value_text: 'My Doc', source: 'user' }]}
        onSubmitted={onSubmitted}
      />,
    );

    fireEvent.click(screen.getByTestId('submit-btn'));

    await waitFor(() => {
      expect(vi.mocked(api.submitDocument)).toHaveBeenCalledWith('doc-1');
    });
    expect(onSubmitted).toHaveBeenCalled();
  });

  it('empty value_text counts as missing', () => {
    render(
      <SubmitButton
        docId="doc-1"
        placeholderSchema={schema}
        placeholderValues={[{ placeholder_id: 'p1', value_text: '', source: 'user' }]}
      />,
    );
    expect(screen.getByTestId('submit-btn')).toBeDisabled();
  });

  it('null value_text counts as missing', () => {
    render(
      <SubmitButton
        docId="doc-1"
        placeholderSchema={schema}
        placeholderValues={[{ placeholder_id: 'p1', value_text: null, source: 'user' }]}
      />,
    );
    expect(screen.getByTestId('submit-btn')).toBeDisabled();
  });
});
