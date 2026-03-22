import { useEffect, useMemo } from "react";
import { ManagedUsersSection } from "../../components/ManagedUsersPanel";
import { WorkspaceDataState } from "../../components/WorkspaceDataState";
import { WorkspaceViewFrame } from "../../components/WorkspaceViewFrame";
import { useManagedUsers } from "./useManagedUsers";
import { useAdminCenter } from "./useAdminCenter";
import styles from "./AdminCenterView.module.css";

function formatDate(value?: string): string {
  if (!value) return "-";
  return new Intl.DateTimeFormat("pt-BR", { dateStyle: "short", timeStyle: "short" }).format(new Date(value));
}

export function AdminCenterView() {
  const adminCenter = useAdminCenter();
  const managedUsersApi = useManagedUsers(adminCenter.refresh);
  const selectedManagedUser = useMemo(
    () => managedUsersApi.managedUsers.find((item) => item.userId === managedUsersApi.managedUserForm.userId) ?? null,
    [managedUsersApi.managedUserForm.userId, managedUsersApi.managedUsers],
  );

  useEffect(() => {
    void adminCenter.refresh();
  }, [adminCenter.refresh]);

  useEffect(() => {
    if (!managedUsersApi.managedUserForm.userId) {
      return;
    }
    const current = managedUsersApi.managedUsers.find((item) => item.userId === managedUsersApi.managedUserForm.userId);
    if (!current) {
      return;
    }
    managedUsersApi.setManagedUserForm((previous) => ({
      ...previous,
      displayName: current.displayName,
      email: current.email ?? "",
      isActive: current.isActive,
      mustChangePassword: current.mustChangePassword,
      roles: Array.isArray(current.roles) && current.roles.length > 0 ? current.roles : previous.roles,
      resetPassword: "",
    }));
  }, [managedUsersApi.managedUserForm.userId, managedUsersApi.managedUsers, managedUsersApi.setManagedUserForm]);

  const onlineCount = adminCenter.onlineUsers.length;
  const latestActivity = adminCenter.recentActivities[0]?.occurredAt;
  const latestActivityLabel = latestActivity ? formatDate(latestActivity) : "Sem atividade recente";
  const toInitials = (value: string) => {
    const [first, second] = value.trim().split(/\s+/);
    return [first?.[0], second?.[0]].filter(Boolean).join("").toUpperCase();
  };

  const activityLabel = (action: string) => {
    const lower = action.toLowerCase();
    if (lower.includes("login")) return "LOGIN";
    if (lower.includes("create") || lower.includes("criad")) return "CRIAR";
    if (lower.includes("approve") || lower.includes("aprov")) return "APROVAR";
    if (lower.includes("edit") || lower.includes("update") || lower.includes("atual")) return "EDICAO";
    return "ACAO";
  };

  const activityVariant = (action: string) => {
    const lower = action.toLowerCase();
    if (lower.includes("login")) return "login";
    if (lower.includes("create") || lower.includes("criad")) return "create";
    if (lower.includes("approve") || lower.includes("aprov")) return "approve";
    if (lower.includes("edit") || lower.includes("update") || lower.includes("atual")) return "edit";
    return "default";
  };

  const activityDotClass = (action: string) => {
    const variant = activityVariant(action);
    if (variant === "login") return `${styles.activityDot} ${styles.activityDotWarning}`;
    if (variant === "create") return `${styles.activityDot} ${styles.activityDotSuccess}`;
    if (variant === "approve") return `${styles.activityDot} ${styles.activityDotInfo}`;
    if (variant === "edit") return `${styles.activityDot} ${styles.activityDotCrimson}`;
    return styles.activityDot;
  };

  const activityChipClass = (action: string) => {
    const variant = activityVariant(action);
    if (variant === "login") return `${styles.activityChip} ${styles.activityChipWarning}`;
    if (variant === "create") return `${styles.activityChip} ${styles.activityChipSuccess}`;
    if (variant === "approve") return `${styles.activityChip} ${styles.activityChipInfo}`;
    if (variant === "edit") return `${styles.activityChip} ${styles.activityChipCrimson}`;
    return styles.activityChip;
  };

  return (
    <WorkspaceViewFrame
      testId="admin-center"
      kicker="IAM + Observabilidade"
      title="Central do Admin"
      description="Controle de usuarios, presenca online e ultimas atividades do sistema."
    >
      <div className={styles.shell}>
        <WorkspaceDataState
          loadState={adminCenter.loadState}
          isEmpty={adminCenter.users.length === 0}
          emptyTitle="Nenhum usuario interno cadastrado"
          emptyDescription="Crie o primeiro usuario para iniciar a administracao de acesso."
          loadingLabel="Atualizando base administrativa"
          onRetry={adminCenter.refresh}
        />

        <section className={styles.headerRow}>
          <div className={styles.liveBadge}>
            <span className={styles.liveDot} />
            Sistema ativo
          </div>
          <span className={styles.headerMeta}>Atualizado {latestActivityLabel}</span>
        </section>

        <section className={styles.summary}>
          <div className={styles.summaryCard}>
            <div className={styles.summaryTop}>
              <div className={`${styles.kpiIcon} ${styles.kpiIconGreen}`}>
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <circle cx="8" cy="5" r="3" />
                  <path d="M2 14c0-3.3 2.7-6 6-6s6 2.7 6 6" />
                </svg>
              </div>
              <span className={styles.kpiTrend}>{onlineCount} ativos</span>
            </div>
            <span className={styles.kpiLabel}>Usuarios online agora</span>
            <strong className={styles.kpiValue}>{onlineCount}</strong>
            <span className={styles.kpiSub}>de {adminCenter.users.length} usuarios</span>
          </div>
          <div className={styles.summaryCard}>
            <div className={styles.summaryTop}>
              <div className={`${styles.kpiIcon} ${styles.kpiIconAmber}`}>
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <circle cx="8" cy="8" r="6.5" />
                  <path d="M8 4v4l3 2" />
                </svg>
              </div>
              <span className={styles.kpiTrend}>hoje</span>
            </div>
            <span className={styles.kpiLabel}>Ultima atividade</span>
            <strong className={styles.kpiValueSmall}>{latestActivityLabel}</strong>
            <span className={styles.kpiSub}>auditoria recente</span>
          </div>
          <div className={styles.summaryCard}>
            <div className={styles.summaryTop}>
              <div className={`${styles.kpiIcon} ${styles.kpiIconRed}`}>
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <rect x="1" y="3" width="14" height="10" rx="1.5" />
                  <path d="M5 3V2M11 3V2M1 7h14" />
                </svg>
              </div>
              <span className={styles.kpiTrend}>{adminCenter.users.length} total</span>
            </div>
            <span className={styles.kpiLabel}>Total de usuarios</span>
            <strong className={styles.kpiValue}>{adminCenter.users.length}</strong>
            <span className={styles.kpiSub}>base completa</span>
          </div>
        </section>

        <section className={styles.grid}>
          <div className={styles.panel}>
            <div className={styles.panelHeader}>
              <div>
                <p className={styles.kicker}>Presenca</p>
                <h2 className={styles.panelTitle}>Usuarios online</h2>
              </div>
              <span className={styles.panelBadge}>{onlineCount} ativos</span>
            </div>
            {onlineCount === 0 ? (
              <p className={styles.empty}>Nenhum usuario online agora.</p>
            ) : (
              <ul className={styles.list}>
                {adminCenter.onlineUsers.map((item, index) => (
                  <li key={item.userId} className={styles.listItem} style={{ animationDelay: `${index * 0.06}s` }}>
                    <span className={styles.avatar}>{toInitials(item.displayName)}</span>
                    <div className={styles.listMeta}>
                      <strong className={styles.listTitle}>{item.displayName}</strong>
                      <small className={styles.listSub}>{item.username}</small>
                    </div>
                    <span className={styles.listTime}>{formatDate(item.lastSeenAt)}</span>
                    <span className={styles.onlinePip} />
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className={styles.panel}>
            <div className={styles.panelHeader}>
              <div>
                <p className={styles.kicker}>Auditoria</p>
                <h2 className={styles.panelTitle}>Ultimas atividades</h2>
              </div>
              <span className={styles.panelBadge}>Ultimos 10</span>
            </div>
            {adminCenter.recentActivities.length === 0 ? (
              <p className={styles.empty}>Nenhuma atividade registrada.</p>
            ) : (
              <ul className={styles.list}>
                {adminCenter.recentActivities.map((item, index) => (
                  <li key={item.id} className={styles.listItem} style={{ animationDelay: `${index * 0.06}s` }}>
                    <span className={activityDotClass(item.action)} />
                    <div className={styles.listMeta}>
                      <strong className={styles.listTitle}>{item.action}</strong>
                      <small className={styles.listSub}>{item.actorId} • {item.resourceType}</small>
                    </div>
                    <span className={styles.listTime}>{formatDate(item.occurredAt)}</span>
                    <span className={activityChipClass(item.action)}>{activityLabel(item.action)}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </section>

        <ManagedUsersSection
          loadState={adminCenter.loadState}
          userForm={managedUsersApi.userForm}
          managedUserForm={managedUsersApi.managedUserForm}
          managedUsers={managedUsersApi.managedUsers}
          selectedManagedUser={selectedManagedUser}
          formatDate={formatDate}
          onRefreshWorkspace={adminCenter.refresh}
          onUserFormChange={managedUsersApi.setUserForm}
          onManagedUserFormChange={managedUsersApi.setManagedUserForm}
          onCreateUser={managedUsersApi.handleCreateUser}
          onSelectManagedUser={managedUsersApi.selectManagedUser}
          onToggleRole={managedUsersApi.toggleManagedUserRole}
          onSaveManagedUser={managedUsersApi.handleSaveManagedUser}
          onAdminResetPassword={managedUsersApi.handleAdminResetPassword}
          onUnlockManagedUser={managedUsersApi.handleUnlockManagedUser}
        />
      </div>
    </WorkspaceViewFrame>
  );
}

export default AdminCenterView;
