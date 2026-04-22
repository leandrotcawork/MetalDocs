import { type FormEvent, useCallback, useEffect, useMemo, useState } from 'react';

import { createRoute, deactivateRoute, listRoutes, updateRoute } from '../api/approvalApi';
import type { DriftPolicy, QuorumKind, Route, RouteStage } from '../api/approvalTypes';
import styles from './RouteAdminPage.module.css';

interface StageDraft {
  label: string;
  membersText: string;
  quorumKind: QuorumKind;
  m: string;
  driftPolicy: DriftPolicy;
}

interface RouteDraft {
  name: string;
  profileCode: string;
  stages: StageDraft[];
}

function toLocalDate(iso: string): string {
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? '-' : date.toLocaleDateString('pt-BR');
}

function parseMembers(membersText: string): string[] {
  return membersText
    .split(',')
    .map((member) => member.trim())
    .filter(Boolean);
}

function defaultStage(): StageDraft {
  return {
    label: '',
    membersText: '',
    quorumKind: 'any_1',
    m: '1',
    driftPolicy: 'auto_cancel',
  };
}

function toDraft(route: Route | null): RouteDraft {
  if (!route) {
    return {
      name: '',
      profileCode: '',
      stages: [defaultStage()],
    };
  }

  return {
    name: route.name,
    profileCode: route.profile_code,
    stages: route.stages.map((stage) => ({
      label: stage.label,
      membersText: stage.members.join(', '),
      quorumKind: stage.quorum_kind,
      m: String(stage.m ?? 1),
      driftPolicy: stage.drift_policy,
    })),
  };
}

function validateDraft(draft: RouteDraft): string | null {
  if (!draft.name.trim()) {
    return 'Informe o nome da rota.';
  }
  if (!draft.profileCode.trim()) {
    return 'Informe o código do perfil.';
  }
  if (draft.stages.length === 0) {
    return 'A rota deve possuir ao menos uma etapa.';
  }

  const labels = new Set<string>();

  for (const stage of draft.stages) {
    const label = stage.label.trim();
    if (!label) {
      return 'Toda etapa deve ter nome.';
    }

    const normalized = label.toLocaleLowerCase('pt-BR');
    if (labels.has(normalized)) {
      return 'Nomes de etapa devem ser distintos.';
    }
    labels.add(normalized);

    const members = parseMembers(stage.membersText);
    if (members.length === 0) {
      return `A etapa "${label}" deve possuir ao menos um membro.`;
    }

    if (stage.quorumKind === 'm_of_n') {
      const mValue = Number(stage.m);
      if (!Number.isFinite(mValue) || mValue < 1) {
        return `Na etapa "${label}", informe um valor de M válido.`;
      }
      if (mValue > members.length) {
        return `Na etapa "${label}", M não pode ser maior que o número de membros.`;
      }
    }
  }

  return null;
}

function toRouteStages(draft: RouteDraft): RouteStage[] {
  return draft.stages.map((stage) => {
    const members = parseMembers(stage.membersText);
    const routeStage: RouteStage = {
      label: stage.label.trim(),
      members,
      quorum_kind: stage.quorumKind,
      drift_policy: stage.driftPolicy,
    };

    if (stage.quorumKind === 'm_of_n') {
      routeStage.m = Number(stage.m);
    }

    return routeStage;
  });
}

interface RouteEditorProps {
  route: Route | null;
  saving: boolean;
  onClose: () => void;
  onSubmit: (draft: RouteDraft) => Promise<void>;
}

function RouteEditor({ route, saving, onClose, onSubmit }: RouteEditorProps) {
  const [draft, setDraft] = useState<RouteDraft>(() => toDraft(route));
  const [error, setError] = useState<string | null>(null);

  const modeTitle = route ? 'Editar rota' : 'Criar rota';

  const updateStage = (index: number, update: Partial<StageDraft>) => {
    setDraft((prev) => {
      const nextStages = prev.stages.map((stage, stageIndex) =>
        stageIndex === index ? { ...stage, ...update } : stage,
      );
      return { ...prev, stages: nextStages };
    });
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError(null);

    const validationError = validateDraft(draft);
    if (validationError) {
      setError(validationError);
      return;
    }

    try {
      await onSubmit(draft);
    } catch (_error) {
      setError('Erro ao salvar rota. Tente novamente.');
    }
  };

  return (
    <div className={styles.overlay}>
      <div className={styles.modal} role="dialog" aria-modal="true" aria-label={modeTitle}>
        <h2 className={styles.modalTitle}>{modeTitle}</h2>

        {error ? (
          <div className={styles.errorBox} role="alert">
            {error}
          </div>
        ) : null}

        <form onSubmit={handleSubmit} className={styles.form}>
          <label className={styles.fieldLabel} htmlFor="route-name">
            Nome da rota
          </label>
          <input
            id="route-name"
            className={styles.input}
            value={draft.name}
            onChange={(event) => setDraft((prev) => ({ ...prev, name: event.target.value }))}
            disabled={saving}
          />

          <label className={styles.fieldLabel} htmlFor="route-profile-code">
            Código do perfil
          </label>
          <input
            id="route-profile-code"
            className={styles.input}
            value={draft.profileCode}
            onChange={(event) => setDraft((prev) => ({ ...prev, profileCode: event.target.value }))}
            disabled={saving || Boolean(route)}
          />

          <section className={styles.stageSection}>
            <div className={styles.stageSectionHeader}>
              <h3>Etapas</h3>
              <button
                type="button"
                className={styles.ghostButton}
                onClick={() =>
                  setDraft((prev) => ({
                    ...prev,
                    stages: [...prev.stages, defaultStage()],
                  }))
                }
                disabled={saving}
              >
                Adicionar etapa
              </button>
            </div>

            {draft.stages.map((stage, index) => (
              <article className={styles.stageCard} key={`stage-${index}`}>
                <div className={styles.stageHeader}>
                  <strong>Etapa {index + 1}</strong>
                  <button
                    type="button"
                    className={styles.linkButton}
                    onClick={() =>
                      setDraft((prev) => ({
                        ...prev,
                        stages: prev.stages.filter((_, stageIndex) => stageIndex !== index),
                      }))
                    }
                    disabled={saving || draft.stages.length === 1}
                  >
                    Remover
                  </button>
                </div>

                <label className={styles.fieldLabel} htmlFor={`stage-label-${index}`}>
                  Nome da etapa {index + 1}
                </label>
                <input
                  id={`stage-label-${index}`}
                  className={styles.input}
                  value={stage.label}
                  onChange={(event) => updateStage(index, { label: event.target.value })}
                  disabled={saving}
                />

                <label className={styles.fieldLabel} htmlFor={`stage-members-${index}`}>
                  Membros da etapa {index + 1}
                </label>
                <input
                  id={`stage-members-${index}`}
                  className={styles.input}
                  value={stage.membersText}
                  onChange={(event) => updateStage(index, { membersText: event.target.value })}
                  disabled={saving}
                />

                <label className={styles.fieldLabel} htmlFor={`stage-quorum-${index}`}>
                  Quórum da etapa {index + 1}
                </label>
                <select
                  id={`stage-quorum-${index}`}
                  className={styles.input}
                  value={stage.quorumKind}
                  onChange={(event) =>
                    updateStage(index, {
                      quorumKind: event.target.value as QuorumKind,
                    })
                  }
                  disabled={saving}
                >
                  <option value="any_1">any_1</option>
                  <option value="all_of">all_of</option>
                  <option value="m_of_n">m_of_n</option>
                </select>

                {stage.quorumKind === 'm_of_n' ? (
                  <>
                    <label className={styles.fieldLabel} htmlFor={`stage-m-${index}`}>
                      M da etapa {index + 1}
                    </label>
                    <input
                      id={`stage-m-${index}`}
                      className={styles.input}
                      type="number"
                      min={1}
                      step={1}
                      value={stage.m}
                      onChange={(event) => updateStage(index, { m: event.target.value })}
                      disabled={saving}
                    />
                  </>
                ) : null}

                <label className={styles.fieldLabel} htmlFor={`stage-drift-policy-${index}`}>
                  Política de drift da etapa {index + 1}
                </label>
                <select
                  id={`stage-drift-policy-${index}`}
                  className={styles.input}
                  value={stage.driftPolicy}
                  onChange={(event) =>
                    updateStage(index, {
                      driftPolicy: event.target.value as DriftPolicy,
                    })
                  }
                  disabled={saving}
                >
                  <option value="auto_cancel">auto_cancel</option>
                  <option value="alert_only">alert_only</option>
                  <option value="none">none</option>
                </select>
              </article>
            ))}
          </section>

          <div className={styles.actions}>
            <button type="button" className={styles.secondaryButton} onClick={onClose} disabled={saving}>
              Cancelar
            </button>
            <button type="submit" className={styles.primaryButton} disabled={saving}>
              {saving ? 'Salvando...' : 'Salvar rota'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export function RouteAdminPage() {
  const [routes, setRoutes] = useState<Route[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showEditor, setShowEditor] = useState(false);
  const [editingRoute, setEditingRoute] = useState<Route | null>(null);
  const [confirmDeactivate, setConfirmDeactivate] = useState<Route | null>(null);
  const [saving, setSaving] = useState(false);

  const fetchRoutes = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await listRoutes();
      setRoutes(response.routes);
    } catch (_error) {
      setError('Erro ao carregar rotas. Tente novamente.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchRoutes();
  }, [fetchRoutes]);

  const canDeactivate = useMemo(() => Boolean(confirmDeactivate), [confirmDeactivate]);

  const closeEditor = () => {
    setShowEditor(false);
    setEditingRoute(null);
  };

  const handleSaveRoute = async (draft: RouteDraft): Promise<void> => {
    setSaving(true);

    const stages = toRouteStages(draft);

    try {
      if (editingRoute) {
        await updateRoute(editingRoute.id, {
          name: draft.name.trim(),
          stages,
        });
      } else {
        await createRoute({
          name: draft.name.trim(),
          profile_code: draft.profileCode.trim(),
          stages,
        });
      }

      closeEditor();
      await fetchRoutes();
    } finally {
      setSaving(false);
    }
  };

  const handleDeactivate = async () => {
    if (!confirmDeactivate) {
      return;
    }

    try {
      await deactivateRoute(confirmDeactivate.id);
      setConfirmDeactivate(null);
      await fetchRoutes();
    } catch (_error) {
      setError('Erro ao desativar rota. Tente novamente.');
    }
  };

  return (
    <section className={styles.page}>
      <header className={styles.header}>
        <h1>Administração de Rotas</h1>
        <button
          type="button"
          className={styles.primaryButton}
          onClick={() => {
            setEditingRoute(null);
            setShowEditor(true);
          }}
        >
          Nova rota
        </button>
      </header>

      {error ? (
        <div className={styles.errorBox} role="alert">
          {error}
        </div>
      ) : null}

      {loading ? <p className={styles.state}>Carregando rotas...</p> : null}

      {!loading ? (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Nome</th>
                <th>Etapas</th>
                <th>Ativo</th>
                <th>Criado em</th>
                <th>Ações</th>
              </tr>
            </thead>
            <tbody>
              {routes.map((route) => {
                const editDisabled = !route.active;
                const editTooltip = editDisabled
                  ? 'Rota referenciada por instância ativa; crie uma nova versão'
                  : undefined;

                return (
                  <tr key={route.id}>
                    <td>{route.name}</td>
                    <td>{route.stages.length} etapa(s)</td>
                    <td>
                      <span className={route.active ? styles.badgeActive : styles.badgeInactive}>
                        {route.active ? 'Ativa' : 'Inativa'}
                      </span>
                    </td>
                    <td>{toLocalDate(route.created_at)}</td>
                    <td className={styles.rowActions}>
                      <button
                        type="button"
                        className={styles.secondaryButton}
                        aria-label={`Editar ${route.name}`}
                        disabled={editDisabled}
                        title={editTooltip}
                        onClick={() => {
                          setEditingRoute(route);
                          setShowEditor(true);
                        }}
                      >
                        Editar
                      </button>
                      <button
                        type="button"
                        className={styles.warnButton}
                        aria-label={`Desativar ${route.name}`}
                        onClick={() => setConfirmDeactivate(route)}
                      >
                        Desativar
                      </button>
                    </td>
                  </tr>
                );
              })}

              {routes.length === 0 ? (
                <tr>
                  <td className={styles.emptyRow} colSpan={5}>
                    Nenhuma rota cadastrada.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      ) : null}

      {showEditor ? (
        <RouteEditor route={editingRoute} saving={saving} onClose={closeEditor} onSubmit={handleSaveRoute} />
      ) : null}

      {confirmDeactivate ? (
        <div className={styles.overlay}>
          <div
            className={styles.confirmDialog}
            role="dialog"
            aria-modal="true"
            aria-label="Confirmar desativação"
          >
            <h2>Confirmar desativação</h2>
            <p>
              Deseja desativar a rota <strong>{confirmDeactivate.name}</strong>?
            </p>
            <div className={styles.actions}>
              <button
                type="button"
                className={styles.secondaryButton}
                onClick={() => setConfirmDeactivate(null)}
              >
                Cancelar
              </button>
              <button
                type="button"
                className={styles.warnButton}
                disabled={!canDeactivate}
                onClick={() => void handleDeactivate()}
              >
                Confirmar
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  );
}

