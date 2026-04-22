import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import * as approvalApi from '../api/approvalApi';
import { RouteAdminPage } from './RouteAdminPage';

vi.mock('../api/approvalApi');

describe('RouteAdminPage', () => {
  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('list render - shows route name and stage count', async () => {
    vi.mocked(approvalApi.listRoutes).mockResolvedValue({
      total: 1,
      routes: [
        {
          id: 'route-1',
          name: 'Rota Jurídica',
          tenant_id: 'tenant-1',
          profile_code: 'JUR',
          stages: [
            {
              label: 'Revisão',
              members: ['ana'],
              quorum_kind: 'any_1',
              drift_policy: 'auto_cancel',
            },
            {
              label: 'Aprovação',
              members: ['carlos'],
              quorum_kind: 'all_of',
              drift_policy: 'none',
            },
          ],
          active: true,
          created_at: '2026-04-20T10:00:00.000Z',
          updated_at: '2026-04-20T10:00:00.000Z',
        },
      ],
    });

    render(<RouteAdminPage />);

    await waitFor(() => {
      expect(screen.getByText('Rota Jurídica')).toBeTruthy();
      expect(screen.getByText('2 etapa(s)')).toBeTruthy();
    });
  });

  it('create opens editor modal', async () => {
    vi.mocked(approvalApi.listRoutes).mockResolvedValue({ total: 0, routes: [] });

    render(<RouteAdminPage />);

    await screen.findByText('Administração de Rotas');
    fireEvent.click(screen.getByRole('button', { name: 'Nova rota' }));

    expect(screen.getByRole('dialog', { name: 'Criar rota' })).toBeTruthy();
  });

  it('edit blocked when route is inactive', async () => {
    vi.mocked(approvalApi.listRoutes).mockResolvedValue({
      total: 1,
      routes: [
        {
          id: 'route-2',
          name: 'Rota Inativa',
          tenant_id: 'tenant-1',
          profile_code: 'FIN',
          stages: [
            {
              label: 'Financeiro',
              members: ['joao'],
              quorum_kind: 'any_1',
              drift_policy: 'alert_only',
            },
          ],
          active: false,
          created_at: '2026-04-20T10:00:00.000Z',
          updated_at: '2026-04-20T10:00:00.000Z',
        },
      ],
    });

    render(<RouteAdminPage />);

    const editButton = await screen.findByRole('button', { name: 'Editar Rota Inativa' });
    expect((editButton as HTMLButtonElement).disabled).toBe(true);
    expect(editButton.getAttribute('title')).toBe(
      'Rota referenciada por instância ativa; crie uma nova versão',
    );
  });

  it('deactivate shows confirmation dialog', async () => {
    vi.mocked(approvalApi.listRoutes).mockResolvedValue({
      total: 1,
      routes: [
        {
          id: 'route-3',
          name: 'Rota Operacional',
          tenant_id: 'tenant-1',
          profile_code: 'OPS',
          stages: [
            {
              label: 'Operação',
              members: ['luis'],
              quorum_kind: 'any_1',
              drift_policy: 'auto_cancel',
            },
          ],
          active: true,
          created_at: '2026-04-20T10:00:00.000Z',
          updated_at: '2026-04-20T10:00:00.000Z',
        },
      ],
    });

    render(<RouteAdminPage />);

    fireEvent.click(await screen.findByRole('button', { name: 'Desativar Rota Operacional' }));

    expect(screen.getByRole('dialog', { name: 'Confirmar desativação' })).toBeTruthy();
  });

  it('m_of_n validation - m greater than members count shows error', async () => {
    vi.mocked(approvalApi.listRoutes).mockResolvedValue({ total: 0, routes: [] });
    vi.mocked(approvalApi.createRoute).mockResolvedValue({ route_id: 'route-new' });

    render(<RouteAdminPage />);

    fireEvent.click(await screen.findByRole('button', { name: 'Nova rota' }));

    fireEvent.change(screen.getByLabelText('Nome da rota'), { target: { value: 'Nova rota' } });
    fireEvent.change(screen.getByLabelText('Código do perfil'), { target: { value: 'JUR' } });
    fireEvent.change(screen.getByLabelText('Nome da etapa 1'), { target: { value: 'Jurídico' } });
    fireEvent.change(screen.getByLabelText('Membros da etapa 1'), { target: { value: 'ana,beto' } });
    fireEvent.change(screen.getByLabelText('Quórum da etapa 1'), { target: { value: 'm_of_n' } });
    fireEvent.change(screen.getByLabelText('M da etapa 1'), { target: { value: '3' } });

    fireEvent.click(screen.getByRole('button', { name: 'Salvar rota' }));

    expect(
      screen.getByText('Na etapa "Jurídico", M não pode ser maior que o número de membros.'),
    ).toBeTruthy();
    expect(vi.mocked(approvalApi.createRoute)).not.toHaveBeenCalled();
  });
});

