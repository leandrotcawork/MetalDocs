import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ExportMenu } from '../ExportMenu';

vi.mock('../api/exportsV2', () => ({
  exportPDF: vi.fn(),
  getDocxSignedURL: vi.fn(),
}));

const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);

beforeEach(() => {
  openSpy.mockClear();
  vi.clearAllMocks();
});

describe('ExportMenu', () => {
  it('PDF happy path — status done with cached=false', async () => {
    const { exportPDF } = await import('../api/exportsV2');
    vi.mocked(exportPDF).mockResolvedValueOnce({
      storage_key: 'k',
      signed_url: 'https://example.com/file.pdf',
      composite_hash: 'abc',
      size_bytes: 20480,
      cached: false,
      revision_id: 'rev1',
    });

    render(<ExportMenu documentID="doc1" canExport={true} />);
    fireEvent.click(screen.getByTestId ? screen.getByRole('button', { name: /export pdf/i }) : screen.getByText('Export PDF'));
    await waitFor(() => screen.getByText(/generated/i));
    expect(screen.getByTestId ? screen.getByRole('button', { name: /export pdf/i }) : screen.getByText('Export PDF')).toBeTruthy();
    const status = document.querySelector('[data-export-status="done"]');
    expect(status).toBeTruthy();
    expect(status?.getAttribute('data-export-cached')).toBe('false');
  });

  it('PDF cache hit — displays Cached', async () => {
    const { exportPDF } = await import('../api/exportsV2');
    vi.mocked(exportPDF).mockResolvedValueOnce({
      storage_key: 'k',
      signed_url: 'https://example.com/file.pdf',
      composite_hash: 'abc',
      size_bytes: 20480,
      cached: true,
      revision_id: 'rev1',
    });

    render(<ExportMenu documentID="doc1" canExport={true} />);
    fireEvent.click(screen.getByText('Export PDF'));
    await waitFor(() => screen.getByText(/cached/i));
    const status = document.querySelector('[data-export-status="done"]');
    expect(status?.getAttribute('data-export-cached')).toBe('true');
  });

  it('429 — shows rate limited message with retry seconds', async () => {
    const { exportPDF } = await import('../api/exportsV2');
    vi.mocked(exportPDF).mockRejectedValueOnce(
      Object.assign(new Error('http_429'), { status: 429, body: { retry_after_seconds: 15 } }),
    );

    render(<ExportMenu documentID="doc1" canExport={true} />);
    fireEvent.click(screen.getByText('Export PDF'));
    await waitFor(() => screen.getByRole('alert'));
    expect(screen.getByRole('alert').textContent).toMatch(/retry in 15s/i);
    expect(document.querySelector('[data-export-status="rate_limited"]')).toBeTruthy();
  });

  it('502 — shows unavailable error message', async () => {
    const { exportPDF } = await import('../api/exportsV2');
    vi.mocked(exportPDF).mockRejectedValueOnce(
      Object.assign(new Error('http_502'), { status: 502 }),
    );

    render(<ExportMenu documentID="doc1" canExport={true} />);
    fireEvent.click(screen.getByText('Export PDF'));
    await waitFor(() => screen.getByRole('alert'));
    expect(screen.getByRole('alert').textContent).toMatch(/unavailable/i);
  });

  it('canExport=false — both buttons disabled, no network call on click', () => {
    render(<ExportMenu documentID="doc1" canExport={false} />);
    const docxBtn = screen.getByText('Download .docx').closest('button')!;
    const pdfBtn = screen.getByText('Export PDF').closest('button')!;
    expect(docxBtn).toBeDisabled();
    expect(pdfBtn).toBeDisabled();
    fireEvent.click(pdfBtn);
    expect(openSpy).not.toHaveBeenCalled();
  });
});
