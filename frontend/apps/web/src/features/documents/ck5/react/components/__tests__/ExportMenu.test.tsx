import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { ExportMenu } from '../ExportMenu';

vi.mock('../../../persistence/exportApi', () => ({
  triggerExport: vi.fn().mockResolvedValue(undefined),
  clientPrint: vi.fn(),
  ExportError: class ExportError extends Error {
    constructor(
      public status: number,
      msg?: string,
    ) {
      super(msg);
    }
  },
}));

describe('ExportMenu', () => {
  it('renders 3 buttons', () => {
    render(<ExportMenu docId="doc1" editorHtml="<p>Hi</p>" />);
    expect(screen.getByText('Export DOCX')).toBeTruthy();
    expect(screen.getByText('Export PDF')).toBeTruthy();
    expect(screen.getByText('Print Preview')).toBeTruthy();
  });

  it('buttons are disabled when editorHtml is null', () => {
    render(<ExportMenu docId="doc1" editorHtml={null} />);
    const buttons = screen.getAllByRole('button');
    buttons.forEach((btn) => expect(btn).toBeDisabled());
  });

  it('clicking Export DOCX calls triggerExport with (docId, "docx")', async () => {
    const { triggerExport } = await import('../../../persistence/exportApi');
    render(<ExportMenu docId="doc123" editorHtml="<p>Hi</p>" />);
    fireEvent.click(screen.getByText('Export DOCX'));
    expect(triggerExport).toHaveBeenCalledWith('doc123', 'docx');
  });

  it('shows error alert when export fails', async () => {
    const { triggerExport, ExportError } = await import('../../../persistence/exportApi');
    vi.mocked(triggerExport).mockRejectedValueOnce(new ExportError(500));
    render(<ExportMenu docId="doc1" editorHtml="<p>Hi</p>" />);
    fireEvent.click(screen.getByText('Export DOCX'));
    await screen.findByRole('alert');
  });
});
