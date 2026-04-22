import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { ApprovalInstance } from '../api/approvalTypes';
import { ApprovalTimelinePanel } from './ApprovalTimelinePanel';

function makeInstance(): ApprovalInstance {
  return {
    id: 'inst-1',
    document_id: 'doc-1',
    route_id: 'route-1',
    status: 'in_progress',
    submitted_by: 'maria',
    submitted_at: '2026-04-15T10:00:00.000Z',
    stages: [
      {
        id: 'stage-1',
        stage_index: 0,
        label: 'Revisão Técnica',
        status: 'active',
        signoffs: [
          {
            id: 'sign-1',
            actor_user_id: 'joao',
            decision: 'approve',
            signature_method: 'password_reauth',
            signed_at: '2026-04-15T11:00:00.000Z',
            reason: 'Tudo certo',
          },
        ],
      },
    ],
  };
}

describe('ApprovalTimelinePanel', () => {
  it('loading state shows skeleton', () => {
    render(<ApprovalTimelinePanel instance={null} loading error={null} />);
    expect(screen.getByText('Carregando timeline...')).toBeTruthy();
  });

  it('empty instance shows empty state', () => {
    render(<ApprovalTimelinePanel instance={null} loading={false} />);
    expect(screen.getByText('Nenhum evento de aprovação registrado.')).toBeTruthy();
  });

  it('renders submitted by and actor name', () => {
    render(<ApprovalTimelinePanel instance={makeInstance()} loading={false} />);
    expect(screen.getByText(/maria/)).toBeTruthy();
    expect(screen.getByText(/joao/)).toBeTruthy();
  });

  it('renders stage with signoff decision', () => {
    render(<ApprovalTimelinePanel instance={makeInstance()} loading={false} />);
    expect(screen.getByText('Revisão Técnica')).toBeTruthy();
    expect(screen.getByText(/Aprovou/)).toBeTruthy();
  });

  it('error state renders with message', () => {
    const onRetry = vi.fn();
    render(
      <ApprovalTimelinePanel
        instance={null}
        loading={false}
        error="Falha ao carregar timeline."
        onRetry={onRetry}
      />,
    );

    expect(screen.getByText('Falha ao carregar timeline.')).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Tentar novamente' }));
    expect(onRetry).toHaveBeenCalledOnce();
  });
});
