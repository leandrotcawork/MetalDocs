import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { DocumentsHubView } from '../DocumentsHubView';
import type { DocumentListItem } from '../../../lib.types';

function makeDoc(status: string): DocumentListItem {
  return {
    documentId: 'doc-abc',
    title: 'Test Doc',
    documentCode: 'DC-001',
    documentProfile: 'DC',
    documentType: 'DC',
    documentSequence: 1,
    profileSchemaVersion: 1,
    status,
    ownerId: 'user-1',
    processArea: 'quality',
    department: 'QA',
    businessUnit: '',
    classification: '',
    subject: '',
    tags: [],
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    effectiveAt: null,
    expiryAt: null,
    hasContent: false,
  } as unknown as DocumentListItem;
}

const baseProps = {
  view: 'library' as const,
  loadState: 'ready' as const,
  currentUserId: 'user-1',
  managedUsers: [],
  documents: [],
  documentProfiles: [],
  processAreas: [],
  selectedProfileGovernance: null,
  searchQuery: '',
  formatDate: (v?: string) => v ?? '-',
  onSearchQueryChange: vi.fn(),
  onCreateDocument: vi.fn(),
  onOpenDocument: vi.fn(),
  onOpenDocumentForHub: vi.fn(),
};

function renderDetail(status: string) {
  const doc = makeDoc(status);
  return render(
    <MemoryRouter initialEntries={['/documents/doc/doc-abc']}>
      <DocumentsHubView
        {...baseProps}
        selectedDocument={doc}
      />
    </MemoryRouter>,
  );
}

describe('DocumentsHubView — Edit button', () => {
  it('shows Editar button for DRAFT (uppercase) documents', () => {
    renderDetail('DRAFT');
    expect(screen.getByRole('button', { name: /editar/i })).toBeTruthy();
  });

  it('shows Editar button for draft (lowercase) documents', () => {
    renderDetail('draft');
    expect(screen.getByRole('button', { name: /editar/i })).toBeTruthy();
  });

  it('does not show Editar button for non-draft documents', () => {
    renderDetail('APPROVED');
    expect(screen.queryByRole('button', { name: /editar/i })).toBeNull();
  });
});
